package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"catalog/internal/dto"
	"catalog/internal/handler"
	"catalog/internal/model"
	"catalog/internal/repository"
	"catalog/internal/service"

	"github.com/gin-gonic/gin"
)

const (
	adminCarsPath  = "/api/v1/catalog/admin/cars"
	publicCarsPath = "/api/v1/catalog/cars"
	secondCarKey   = "catalog/images/zyxwvutsrqponmlkjihgfedcba.webp"
)

type carTestImages struct{}

func (carTestImages) ImageExists(ctx context.Context, key string) (bool, error) {
	if key == secondCarKey {
		return true, nil
	}
	return (testImages{}).ImageExists(ctx, key)
}

func newCarTestRouter(t *testing.T, repo *repository.CarRepository) *gin.Engine {
	t.Helper()
	cars := handler.NewCarHandler(service.NewCarService(repo, carTestImages{}, "https://assets.example.test"))
	return newCatalogTestRouter(t, handler.NewBrandHandler(nil), handler.NewBannerHandler(nil), cars)
}

func TestCarAuthorizationAndInvalidRequests(t *testing.T) {
	router := newCarTestRouter(t, nil)
	for _, test := range []struct {
		method, path, token, body string
		status                    int
	}{
		{"GET", adminCarsPath, "", "", 401},
		{"POST", adminCarsPath, "CUSTOMER", "{}", 403},
		{"PATCH", adminCarsPath + "/" + missingBrandID, "CUSTOMER", "{}", 403},
		{"GET", adminCarsPath + "/bad", "ADMIN", "", 400},
		{"POST", adminCarsPath + "/" + missingBrandID + "/images", "ADMIN", "{}", 404},
		{"DELETE", adminCarsPath + "/" + missingBrandID + "/images/" + missingBrandID, "ADMIN", "", 404},
		{"POST", adminCarsPath, "OWNER", `{"is_featured":true}`, 400},
		{"POST", adminCarsPath, "OWNER", `{"status":"active"}`, 400},
		{"PATCH", adminCarsPath + "/" + missingBrandID, "ADMIN", `{"model":null}`, 400},
		{"PATCH", adminCarsPath + "/" + missingBrandID, "ADMIN", `{"model":" "}`, 400},
		{"PATCH", adminCarsPath + "/" + missingBrandID, "ADMIN", `{"seats":0}`, 400},
		{"PATCH", adminCarsPath + "/" + missingBrandID, "ADMIN", `{"daily_rate_idr":0}`, 400},
		{"PATCH", adminCarsPath + "/" + missingBrandID, "ADMIN", `{}`, 400},
		{"GET", publicCarsPath + "?min_price=2&max_price=1", "", "", 400},
		{"GET", publicCarsPath + "?status=inactive", "", "", 400},
		{"GET", publicCarsPath + "?sort=unsafe", "", "", 400},
		{"GET", publicCarsPath + "?limit=101", "", "", 400},
		{"GET", publicCarsPath + "?brand_id=bad", "", "", 400},
		{"GET", publicCarsPath + "/test/related?limit=13", "", "", 400},
		{"GET", publicCarsPath + "/test/related?limit=%", "", "", 400},
	} {
		catalogRequest(t, router, test.method, test.path, test.token, test.body, test.status)
	}
}

func TestCarPublicationPostgres(t *testing.T) {
	db := catalogTestDatabase(t)
	brands := repository.NewBrandRepository(db)
	toyota := createCarTestBrand(t, brands, "Toyota")
	honda := createCarTestBrand(t, brands, "Honda")
	router := newCarTestRouter(t, repository.NewCarRepository(db))

	req := carTestRequest(toyota.ID, "Fortuner", 600000)
	first := createTestCar(t, router, req)
	adminPath := adminCarsPath + "/" + first.ID
	publicPath := publicCarsPath + "/" + first.Slug
	if first.Status != "inactive" || first.Slug != "toyota-fortuner-vrz-at" || first.Cover == nil || len(first.Images) != 2 {
		t.Fatalf("unexpected created car: %+v", first)
	}
	catalogRequest(t, router, "GET", adminPath, "ADMIN", "", 200)
	catalogRequest(t, router, "GET", publicPath, "", "", 404)
	catalogRequest(t, router, "GET", publicPath+"/related", "", "", 404)
	catalogRequest(t, router, "POST", adminCarsPath, "ADMIN", carJSON(t, req), 409)
	slugCollision := req
	slugCollision.Trim = "VRZ-AT"
	catalogRequest(t, router, "POST", adminCarsPath, "ADMIN", carJSON(t, slugCollision), 409)
	var missing = req
	missing.BrandID = missingBrandID
	catalogRequest(t, router, "POST", adminCarsPath, "ADMIN", carJSON(t, missing), 400)
	// Simulate an inactive car saved before the form required a cover.
	if err := db.Delete(&model.CarImage{}, "id = ?", first.Images[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	catalogRequest(t, router, "PATCH", adminPath, "OWNER", `{"status":"active"}`, 409)
	catalogRequest(t, router, "PATCH", adminPath, "ADMIN", carJSON(t, map[string]any{"status": "active", "cover": req.Cover}), 200)

	var public struct{ Data dto.CarDetail }
	body := catalogRequest(t, router, "GET", publicPath, "", "", 200)
	decodeCarJSON(t, body, &public)
	if len(public.Data.Gallery) != 2 || public.Data.Gallery[0].Label != "Exterior" || public.Data.Gallery[1].Label != "Interior" ||
		public.Data.Cover == nil || public.Data.Cover.URL == nil || *public.Data.Cover.URL != "https://assets.example.test/"+uploadedKey ||
		public.Data.FuelType != "diesel" || strings.Contains(string(body), `"object_key"`) || public.Data.Status != model.CarActive {
		t.Fatalf("unexpected public detail: %s", body)
	}
	invalidCover := dto.CarPhotoInput{ObjectKey: missingKey, AltText: "Missing", Label: "Missing"}
	catalogRequest(t, router, "PATCH", adminPath, "ADMIN", carJSON(t, map[string]any{"cover": invalidCover}), 400)
	body = catalogRequest(t, router, "GET", publicPath, "", "", 200)
	decodeCarJSON(t, body, &public)
	if public.Data.Cover == nil || public.Data.Cover.Label != "Exterior" {
		t.Fatalf("failed attach should preserve cover: %s", body)
	}
	second := createTestCar(t, router, carTestRequest(toyota.ID, "Rush", 500000))
	secondPath := adminCarsPath + "/" + second.ID
	catalogRequest(t, router, "PATCH", secondPath, "ADMIN", `{"model":"fortuner"}`, 409)
	changed := catalogRequest(t, router, "PATCH", adminPath, "OWNER", `{"model":"Fortuner GR","trim":""}`, 200)
	var updated struct{ Data dto.AdminCarDetail }
	decodeCarJSON(t, changed, &updated)
	if updated.Data.Model != "Fortuner GR" || updated.Data.Trim != "" || updated.Data.Slug != first.Slug || updated.Data.DailyRateIDR != 600000 {
		t.Fatalf("patch lost fields or changed slug: %s", changed)
	}

	inactive := model.BrandInactive
	for _, brand := range []model.Brand{toyota, honda} {
		if _, err := brands.Update(context.Background(), brand.ID, model.BrandChanges{Status: &inactive}); err != nil {
			t.Fatal(err)
		}
	}
	catalogRequest(t, router, "POST", adminCarsPath, "ADMIN", carJSON(t, carTestRequest(toyota.ID, "Yaris", 400000)), 400)
	catalogRequest(t, router, "PATCH", adminPath, "ADMIN", `{"brand_id":"`+honda.ID+`"}`, 400)
	catalogRequest(t, router, "GET", publicPath, "", "", 200)
	var options struct{ Data []dto.CarBrandOption }
	decodeCarJSON(t, catalogRequest(t, router, "GET", "/api/v1/catalog/brands", "", "", 200), &options)
	if len(options.Data) != 1 || options.Data[0].ID != toyota.ID {
		t.Fatalf("expected filter to include brand with active cars: %+v", options)
	}

	catalogRequest(t, router, "PATCH", adminPath, "ADMIN", `{"status":"inactive"}`, 200)
	catalogRequest(t, router, "GET", publicPath, "", "", 404)
	catalogRequest(t, router, "GET", publicPath+"/related", "", "", 404)
	var list dto.AdminCarList
	decodeCarJSON(t, catalogRequest(t, router, "GET", adminCarsPath+"?status=inactive&search=toyota", "ADMIN", "", 200), &list)
	if len(list.Data) != 2 || list.Data[0].ID != second.ID || list.Page.Total != 2 {
		t.Fatalf("unexpected admin list order or filters: %+v", list)
	}
}

func TestCarFormSubmissionPostgres(t *testing.T) {
	db := catalogTestDatabase(t)
	brand := createCarTestBrand(t, repository.NewBrandRepository(db), "Toyota")
	router := newCarTestRouter(t, repository.NewCarRepository(db))
	active := model.CarActive

	req := carTestRequest(brand.ID, "Fortuner", 650000)
	req.Status = &active
	withoutGallery := req
	withoutGallery.Gallery = []dto.CarPhotoInput{}
	catalogRequest(t, router, "POST", adminCarsPath, "ADMIN", carJSON(t, withoutGallery), 400)
	withMissingPhoto := req
	withMissingPhoto.Gallery = []dto.CarPhotoInput{{ObjectKey: missingKey, AltText: "Interior", Label: "Interior"}}
	catalogRequest(t, router, "POST", adminCarsPath, "ADMIN", carJSON(t, withMissingPhoto), 400)
	var list dto.AdminCarList
	decodeCarJSON(t, catalogRequest(t, router, "GET", adminCarsPath, "ADMIN", "", 200), &list)
	if list.Page.Total != 0 {
		t.Fatalf("invalid form left a car behind: %+v", list)
	}

	car := createTestCar(t, router, req)
	if car.Status != model.CarActive || len(car.Images) != 2 {
		t.Fatalf("expected a published car with both images: %+v", car)
	}
	catalogRequest(t, router, "GET", publicCarsPath+"/"+car.Slug, "", "", 200)
	path := adminCarsPath + "/" + car.ID
	edit := map[string]any{
		"daily_rate_idr": 675000,
		"cover":          dto.CarPhotoInput{ObjectKey: secondCarKey, AltText: "New cover", Label: "New cover"},
		"gallery":        []dto.CarPhotoInput{{ObjectKey: uploadedKey, AltText: "New gallery", Label: "New gallery"}},
	}
	var response struct{ Data dto.AdminCarDetail }
	decodeCarJSON(t, catalogRequest(t, router, "PATCH", path, "ADMIN", carJSON(t, edit), 200), &response)
	if response.Data.DailyRateIDR != 675000 || len(response.Data.Images) != 2 ||
		response.Data.Images[0].Label != "New cover" || response.Data.Images[1].Label != "New gallery" {
		t.Fatalf("expected fields and photos to change together: %+v", response.Data)
	}

	createTestCar(t, router, carTestRequest(brand.ID, "Rush", 500000))
	edit["model"] = "Rush"
	edit["cover"] = req.Cover
	edit["daily_rate_idr"] = 700000
	catalogRequest(t, router, "PATCH", path, "ADMIN", carJSON(t, edit), 409)
	catalogRequest(t, router, "PATCH", path, "ADMIN", `{"gallery":[]}`, 400)
	decodeCarJSON(t, catalogRequest(t, router, "GET", path, "ADMIN", "", 200), &response)
	if response.Data.Model != "Fortuner" || response.Data.DailyRateIDR != 675000 ||
		len(response.Data.Images) != 2 || response.Data.Images[0].Label != "New cover" {
		t.Fatalf("failed edit changed the saved car: %+v", response.Data)
	}

	const oldUpdatedAt = "2024-01-01T00:00:00Z"
	if err := db.Model(&model.Car{}).Where("id = ?", car.ID).
		UpdateColumn("updated_at", oldUpdatedAt).Error; err != nil {
		t.Fatal(err)
	}
	photoOnly := carJSON(t, map[string]any{"cover": req.Cover})
	decodeCarJSON(t, catalogRequest(t, router, "PATCH", path, "ADMIN", photoOnly, 200), &response)
	if response.Data.UpdatedAt <= oldUpdatedAt || response.Data.Images[0].Label != "Exterior" {
		t.Fatalf("photo-only edit should update the car timestamp and cover: %+v", response.Data)
	}
}

func TestCarGalleryOrderPostgres(t *testing.T) {
	db := catalogTestDatabase(t)
	brand := createCarTestBrand(t, repository.NewBrandRepository(db), "Toyota")
	router := newCarTestRouter(t, repository.NewCarRepository(db))
	active := model.CarActive
	req := carTestRequest(brand.ID, "Fortuner", 650000)
	req.Status = &active
	req.Gallery = append(req.Gallery, dto.CarPhotoInput{
		ObjectKey: uploadedKey, AltText: "Dashboard view", Label: "Dashboard",
	})
	car := createTestCar(t, router, req)
	assertAdminOrder := func(item dto.AdminCarDetail, want []string) {
		t.Helper()
		labels := make([]string, 0, len(item.Images))
		for _, image := range item.Images {
			labels = append(labels, image.Label)
		}
		if !slices.Equal(labels, want) {
			t.Fatalf("gallery order = %v, want %v", labels, want)
		}
	}
	assertAdminOrder(car, []string{"Exterior", "Interior", "Dashboard"})

	req.Gallery = append(req.Gallery, dto.CarPhotoInput{
		ObjectKey: secondCarKey, AltText: "Rear view", Label: "Rear",
	})
	path := adminCarsPath + "/" + car.ID
	want := []string{"Exterior", "Interior", "Dashboard", "Rear"}
	var result struct{ Data dto.AdminCarDetail }
	edit := carJSON(t, map[string]any{"gallery": req.Gallery})
	decodeCarJSON(t, catalogRequest(t, router, "PATCH", path, "ADMIN", edit, 200), &result)
	assertAdminOrder(result.Data, want)

	// The last photo stays last even if its timestamp would sort before the other photos.
	if err := db.Model(&model.CarImage{}).Where("car_id = ? AND label = ?", car.ID, "Rear").
		UpdateColumn("created_at", "2000-01-01T00:00:00Z").Error; err != nil {
		t.Fatal(err)
	}
	decodeCarJSON(t, catalogRequest(t, router, "GET", path, "ADMIN", "", 200), &result)
	assertAdminOrder(result.Data, want)

	var public struct{ Data dto.CarDetail }
	decodeCarJSON(t, catalogRequest(t, router, "GET", publicCarsPath+"/"+car.Slug, "", "", 200), &public)
	labels := make([]string, 0, len(public.Data.Gallery))
	for _, image := range public.Data.Gallery {
		labels = append(labels, image.Label)
	}
	if !slices.Equal(labels, want) {
		t.Fatalf("public gallery order = %v, want %v", labels, want)
	}
}

func TestCarFiltersPostgres(t *testing.T) {
	db := catalogTestDatabase(t)
	brands := repository.NewBrandRepository(db)
	toyota := createCarTestBrand(t, brands, "Toyota")
	honda := createCarTestBrand(t, brands, "Honda")
	router := newCarTestRouter(t, repository.NewCarRepository(db))

	var cars []dto.AdminCarDetail
	for index, rate := range []int64{400000, 400001, 700000, 700001} {
		req := carTestRequest(toyota.ID, []string{`Avanza%_\`, "Raize", "Rush", "Fortuner"}[index], rate)
		if index == 0 {
			req.Seats = 5
		}
		if index == 2 {
			req.Transmission = "manual"
		}
		if index == 3 {
			req.BrandID = honda.ID
		}
		car := createTestCar(t, router, req)
		path := adminCarsPath + "/" + car.ID
		catalogRequest(t, router, "PATCH", path, "ADMIN", `{"status":"active"}`, 200)
		cars = append(cars, car)
	}
	createTestCar(t, router, carTestRequest(honda.ID, "Brio", 450000))
	for _, test := range []struct {
		query string
		want  []int64
		total int64
	}{
		{"?max_price=400000", []int64{400000}, 1},
		{"?min_price=400001&max_price=700000&sort=price-asc", []int64{400001, 700000}, 2},
		{"?min_price=700001", []int64{700001}, 1},
		{"?brand_id=" + toyota.ID + "&seats=7&category=suv&transmission=manual&search=VRZ", []int64{700000}, 1},
		{"?seats=5", []int64{400000}, 1},
		{"?search=%25_%5C", []int64{400000}, 1},
		{"?search=toyo&sort=name-asc", []int64{400000, 400001, 700000}, 3},
		{"?sort=price-desc&page=2&limit=2", []int64{400001, 400000}, 4},
		{"?page=3&limit=2", []int64{}, 4},
		{"?search=missing", []int64{}, 0},
	} {
		var result dto.CarList
		body := catalogRequest(t, router, "GET", publicCarsPath+test.query, "", "", 200)
		decodeCarJSON(t, body, &result)
		got := make([]int64, 0, len(result.Data))
		for _, car := range result.Data {
			got = append(got, car.DailyRateIDR)
		}
		if result.Data == nil || !slices.Equal(got, test.want) || result.Page.Total != test.total {
			t.Fatalf("query %s gave %s, want %v with total %d", test.query, body, test.want, test.total)
		}
	}
	related := catalogRequest(t, router, "GET", publicCarsPath+"/"+cars[3].Slug+"/related?limit=2", "", "", 200)
	var result struct{ Data []dto.CarCard }
	decodeCarJSON(t, related, &result)
	if len(result.Data) != 2 || result.Data[0].ID != cars[2].ID || result.Data[1].ID != cars[1].ID {
		t.Fatalf("unexpected related cars: %s", related)
	}
	if err := db.Model(&model.Car{}).Where("id IN ?", []string{cars[0].ID, cars[1].ID}).
		UpdateColumn("created_at", cars[2].CreatedAt).Error; err != nil {
		t.Fatal(err)
	}
	var ordered dto.CarList
	decodeCarJSON(t, catalogRequest(t, router, "GET", publicCarsPath+"?max_price=400001", "", "", 200), &ordered)
	wantIDs := []string{cars[0].ID, cars[1].ID}
	slices.Sort(wantIDs)
	if len(ordered.Data) != 2 || ordered.Data[0].ID != wantIDs[1] || ordered.Data[1].ID != wantIDs[0] {
		t.Fatalf("default sort does not break timestamp ties by descending ID: %+v", ordered)
	}
}

func TestCarCoverReplacementAndActivationPostgres(t *testing.T) {
	db := catalogTestDatabase(t)
	brand := createCarTestBrand(t, repository.NewBrandRepository(db), "Toyota")
	router := newCarTestRouter(t, repository.NewCarRepository(db))
	car := createTestCar(t, router, carTestRequest(brand.ID, "Fortuner", 600000))
	path := adminCarsPath + "/" + car.ID
	activate := httptest.NewRequest("PATCH", path, strings.NewReader(`{"status":"active"}`))
	cover := dto.CarPhotoInput{ObjectKey: secondCarKey, AltText: "New exterior", Label: "New cover"}
	replace := httptest.NewRequest("PATCH", path, strings.NewReader(carJSON(t, map[string]any{"cover": cover})))
	start := make(chan struct{})
	statuses := make(chan int, 2)
	for _, req := range []*http.Request{activate, replace} {
		req.Header.Set("Authorization", "Bearer ADMIN")
		req.Header.Set("Content-Type", "application/json")
		go func() {
			<-start
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			statuses <- response.Code
		}()
	}
	close(start)
	first, second := <-statuses, <-statuses
	if first != 200 || second != 200 {
		t.Fatalf("expected both updates to succeed, got %d and %d", first, second)
	}
	var detail struct{ Data dto.AdminCarDetail }
	decodeCarJSON(t, catalogRequest(t, router, "GET", path, "OWNER", "", 200), &detail)
	if detail.Data.Status != "active" || detail.Data.Cover == nil || detail.Data.Cover.Label != "New cover" || len(detail.Data.Images) != 2 {
		t.Fatalf("inconsistent result after concurrent requests: %+v", detail.Data)
	}
}

func createCarTestBrand(t *testing.T, repo *repository.BrandRepository, name string) model.Brand {
	t.Helper()
	brand, err := repo.Create(context.Background(), model.Brand{Name: name})
	if err != nil {
		t.Fatal(err)
	}
	return brand
}

func carTestRequest(brandID, name string, rate int64) dto.CreateCarRequest {
	return dto.CreateCarRequest{
		BrandID: brandID, Model: name, Trim: "VRZ AT", Category: "suv", BodyType: "suv", Seats: 7,
		Transmission: "automatic", FuelType: "diesel", Description: "Demo car.", DailyRateIDR: rate,
		Cover:   &dto.CarPhotoInput{ObjectKey: uploadedKey, AltText: "Exterior view", Label: "Exterior"},
		Gallery: []dto.CarPhotoInput{{ObjectKey: secondCarKey, AltText: "Interior view", Label: "Interior"}},
	}
}

func createTestCar(t *testing.T, router *gin.Engine, req dto.CreateCarRequest) dto.AdminCarDetail {
	t.Helper()
	var result struct {
		Message string
		Data    dto.AdminCarDetail
	}
	decodeCarJSON(t, catalogRequest(t, router, "POST", adminCarsPath, "OWNER", carJSON(t, req), 201), &result)
	if result.Message != "Car created successfully" || result.Data.ID == "" {
		t.Fatalf("unexpected create result: %+v", result)
	}
	return result.Data
}

func decodeCarJSON(t *testing.T, body []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("invalid response %s: %v", body, err)
	}
}

func carJSON(t *testing.T, value any) string {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}
