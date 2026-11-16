package router

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"catalog/internal/dto"
	"catalog/internal/handler"
	"catalog/internal/model"
	"catalog/internal/repository"
	"catalog/internal/service"
)

const adminBannersPath = "/api/v1/catalog/admin/banners"
const publicBannersPath = "/api/v1/catalog/home/banners"

func TestBannerAuthorizationAndValidation(t *testing.T) {
	banners := handler.NewBannerHandler(service.NewBannerService(nil, testImages{}, ""))
	router := newCatalogTestRouter(t, handler.NewBrandHandler(nil), banners, handler.NewCarHandler(nil), nil)
	catalogRequest(t, router, "GET", adminBannersPath, "", "", 401)
	catalogRequest(t, router, "PUT", adminBannersPath+"/1", "CUSTOMER", "{}", 403)
	catalogRequest(t, router, "DELETE", adminBannersPath+"/1", "CUSTOMER", "", 403)
	catalogRequest(t, router, "GET", adminBannersPath+"/4", "ADMIN", "", 400)
	catalogRequest(t, router, "PUT", adminBannersPath+"/1", "ADMIN", "{}", 400)
}

func TestBannerSlotsPostgres(t *testing.T) {
	db := catalogTestDatabase(t)
	brands := repository.NewBrandRepository(db)
	brand, err := brands.Create(context.Background(), model.Brand{Name: "Toyota"})
	if err != nil {
		t.Fatal(err)
	}
	banners := handler.NewBannerHandler(service.NewBannerService(repository.NewBannerRepository(db), testImages{}, "https://assets.example.test/"))
	router := newCatalogTestRouter(t, handler.NewBrandHandler(nil), banners, handler.NewCarHandler(nil), nil)

	var adminList struct {
		Data []dto.BannerResponse
	}
	initial := catalogRequest(t, router, "GET", adminBannersPath, "OWNER", "", 200)
	if err := json.Unmarshal(initial, &adminList); err != nil {
		t.Fatal(err)
	}
	if len(adminList.Data) != 3 {
		t.Fatalf("expected exactly three slots, got %s", initial)
	}
	for index, item := range adminList.Data {
		if item.Position != index+1 || item.Content != nil {
			t.Fatalf("expected empty slot %d, got %+v", index+1, item)
		}
	}
	empty := catalogRequest(t, router, "GET", publicBannersPath, "", "", 200)
	if string(empty) != `{"data":[]}` {
		t.Fatalf("expected empty public list, got %s", empty)
	}

	saveBody := func(brandID, carModel, slug string) string {
		t.Helper()
		body, err := json.Marshal(dto.UpdateBannerRequest{
			BrandID: brandID, CarModel: carModel, CarSlug: slug,
			HeadlineLine1: "Room for", HeadlineLine2: "bigger plans.",
			Description: "A demo hero banner.", ImageObjectKey: uploadedKey,
		})
		if err != nil {
			t.Fatal(err)
		}
		return string(body)
	}
	catalogRequest(t, router, "PUT", adminBannersPath+"/3", "OWNER", saveBody(brand.ID, "Yaris", "yaris"), 200)
	catalogRequest(t, router, "PUT", adminBannersPath+"/1", "ADMIN", saveBody(brand.ID, "Fortuner", "fortuner-vrz-at"), 200)
	catalogRequest(t, router, "PUT", adminBannersPath+"/1", "ADMIN", saveBody(missingBrandID, "Missing", "missing"), 400)

	var detail struct {
		Data dto.BannerResponse
	}
	detailBody := catalogRequest(t, router, "GET", adminBannersPath+"/1", "ADMIN", "", 200)
	if err := json.Unmarshal(detailBody, &detail); err != nil {
		t.Fatal(err)
	}
	if detail.Data.Content == nil || detail.Data.Content.BrandName != "Toyota" || detail.Data.Content.CarModel != "Fortuner" {
		t.Fatalf("expected saved content and brand in detail, got %s", detailBody)
	}

	var publicList struct {
		Data []dto.PublicBannerResponse
	}
	public := catalogRequest(t, router, "GET", publicBannersPath, "", "", 200)
	if err := json.Unmarshal(public, &publicList); err != nil {
		t.Fatal(err)
	}
	if len(publicList.Data) != 2 || publicList.Data[0].Slug != "fortuner-vrz-at" || publicList.Data[1].Slug != "yaris" ||
		publicList.Data[0].Make != "Toyota" || publicList.Data[0].Headline != [2]string{"Room for", "bigger plans."} ||
		publicList.Data[0].Image == nil || *publicList.Data[0].Image != "https://assets.example.test/"+uploadedKey {
		t.Fatalf("unexpected public hero order or fields: %s", public)
	}

	catalogRequest(t, router, "PUT", adminBannersPath+"/1", "OWNER", saveBody(brand.ID, "Avanza", "avanza"), 200)
	cleared := catalogRequest(t, router, "DELETE", adminBannersPath+"/1", "OWNER", "", 200)
	if err := json.Unmarshal(cleared, &detail); err != nil {
		t.Fatal(err)
	}
	if detail.Data.Position != 1 || detail.Data.Content != nil {
		t.Fatalf("expected slot 1 to stay empty after delete, got %s", cleared)
	}
	finalList := catalogRequest(t, router, "GET", adminBannersPath, "ADMIN", "", 200)
	if err := json.Unmarshal(finalList, &adminList); err != nil {
		t.Fatal(err)
	}
	if len(adminList.Data) != 3 || adminList.Data[0].Content != nil || adminList.Data[2].Content == nil {
		t.Fatalf("unexpected admin list after delete: %s", finalList)
	}
	public = catalogRequest(t, router, "GET", publicBannersPath, "", "", 200)
	if err := json.Unmarshal(public, &publicList); err != nil {
		t.Fatal(err)
	}
	if len(publicList.Data) != 1 || publicList.Data[0].Slug != "yaris" {
		t.Fatalf("expected only slot 3 to remain public, got %s", public)
	}
}

func TestBannerCarReferencePostgres(t *testing.T) {
	db := catalogTestDatabase(t)
	brands := repository.NewBrandRepository(db)
	toyota := createCarTestBrand(t, brands, "Toyota")
	honda := createCarTestBrand(t, brands, "Honda")
	banners := handler.NewBannerHandler(service.NewBannerService(repository.NewBannerRepository(db), testImages{}, "https://assets.example.test"))
	cars := handler.NewCarHandler(service.NewCarService(repository.NewCarRepository(db), carTestImages{}, ""))
	router := newCatalogTestRouter(t, handler.NewBrandHandler(nil), banners, cars, nil)

	active := model.CarActive
	req := carTestRequest(toyota.ID, "Fortuner", 600000)
	req.Status = &active
	selected := createTestCar(t, router, req)
	other := carTestRequest(toyota.ID, "Rush", 500000)
	other.Status = &active
	suggestion := createTestCar(t, router, other)
	unlinked := createTestCar(t, router, carTestRequest(toyota.ID, "Yaris", 450000))

	slot := adminBannersPath + "/1"
	carPath := publicCarsPath + "/" + selected.Slug
	banner := dto.UpdateBannerRequest{
		CarID: selected.ID, HeadlineLine1: "Room for", HeadlineLine2: "bigger plans.",
		Description: "A demo hero banner.", ImageObjectKey: uploadedKey,
	}
	var saved struct{ Data dto.BannerResponse }
	decodeCarJSON(t, catalogRequest(t, router, "PUT", slot, "OWNER", carJSON(t, banner), 200), &saved)
	if saved.Data.Content == nil || saved.Data.Content.CarID == nil || *saved.Data.Content.CarID != selected.ID ||
		saved.Data.Content.CarSlug != selected.Slug || saved.Data.Content.BrandName != "Toyota" {
		t.Fatalf("expected linked car identity, got %+v", saved.Data.Content)
	}

	mixed := banner
	mixed.CarSlug = selected.Slug
	catalogRequest(t, router, "PUT", slot, "ADMIN", carJSON(t, mixed), 400)
	banner.CarID = "invalid"
	catalogRequest(t, router, "PUT", slot, "ADMIN", carJSON(t, banner), 400)
	banner.CarID = missingBrandID
	catalogRequest(t, router, "PUT", slot, "ADMIN", carJSON(t, banner), 404)
	catalogRequest(t, router, "PUT", slot, "CUSTOMER", carJSON(t, banner), 403)

	catalogRequest(t, router, "PATCH", adminCarsPath+"/"+selected.ID, "ADMIN",
		carJSON(t, map[string]any{"status": "inactive", "model": "Fortuner GR", "brand_id": honda.ID}), 200)
	var public struct{ Data []dto.PublicBannerResponse }
	decodeCarJSON(t, catalogRequest(t, router, "GET", publicBannersPath, "", "", 200), &public)
	if len(public.Data) != 1 || public.Data[0].Slug != selected.Slug || public.Data[0].Model != "Fortuner GR" || public.Data[0].Make != "Honda" {
		t.Fatalf("expected banner to follow car details and stay visible: %+v", public.Data)
	}

	var detail struct{ Data dto.CarDetail }
	decodeCarJSON(t, catalogRequest(t, router, "GET", carPath, "", "", 200), &detail)
	if detail.Data.Status != model.CarInactive || detail.Data.ID != selected.ID {
		t.Fatalf("expected detail with inactive status, got %+v", detail.Data)
	}
	catalogRequest(t, router, "GET", publicCarsPath+"/"+unlinked.Slug, "", "", 404)
	var list dto.CarList
	decodeCarJSON(t, catalogRequest(t, router, "GET", publicCarsPath, "", "", 200), &list)
	var related struct{ Data []dto.CarCard }
	decodeCarJSON(t, catalogRequest(t, router, "GET", carPath+"/related", "", "", 200), &related)
	if len(list.Data) != 1 || list.Data[0].ID != suggestion.ID ||
		len(related.Data) != 1 || related.Data[0].ID != suggestion.ID {
		t.Fatal("listing and related results should offer only active cars")
	}

	// Re-saving via the old form keeps the reference when the brand and slug match.
	banner.CarID, banner.BrandID, banner.CarModel, banner.CarSlug = "", honda.ID, "Fortuner GR", selected.Slug
	decodeCarJSON(t, catalogRequest(t, router, "PUT", slot, "ADMIN", carJSON(t, banner), 200), &saved)
	if saved.Data.Content.CarID == nil || *saved.Data.Content.CarID != selected.ID {
		t.Fatalf("expected the legacy form to preserve the link: %+v", saved.Data.Content)
	}

	decodeCarJSON(t, catalogRequest(t, router, "DELETE", slot, "ADMIN", "", 200), &saved)
	if saved.Data.Content != nil {
		t.Fatal("expected the slot to be empty after clearing it")
	}
	catalogRequest(t, router, "GET", carPath, "", "", 404)
	catalogRequest(t, router, "GET", carPath+"/related", "", "", 404)
}

func TestBannerCarReferenceMigrationPostgres(t *testing.T) {
	db := catalogTestDatabase(t)
	brands := repository.NewBrandRepository(db)
	brand := createCarTestBrand(t, brands, "Toyota")
	otherBrand := createCarTestBrand(t, brands, "Honda")
	router := newCarTestRouter(t, repository.NewCarRepository(db))
	car := createTestCar(t, router, carTestRequest(brand.ID, "Fortuner", 600000))

	migrate := func(direction string) {
		t.Helper()
		sql, err := os.ReadFile("../../migrations/000009_link_banners_to_cars." + direction + ".sql")
		if err != nil {
			t.Fatal(err)
		}
		if err := db.Exec(string(sql)).Error; err != nil {
			t.Fatal(err)
		}
	}
	migrate("down")
	for pos, brandID := range map[int]string{1: brand.ID, 2: otherBrand.ID} {
		err := db.Exec(`UPDATE banners SET brand_id=?, car_model='Legacy model', car_slug=?,
			headline_line_1='Room for', headline_line_2='bigger plans.', description='Legacy description',
			image_object_key=? WHERE position=?`, brandID, car.Slug, uploadedKey, pos).Error
		if err != nil {
			t.Fatal(err)
		}
	}
	migrate("up")
	banners, err := repository.NewBannerRepository(db).List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(banners) != 3 || banners[0].CarID == nil || *banners[0].CarID != car.ID ||
		banners[1].CarID != nil || banners[1].CarModel == nil || *banners[1].CarModel != "Legacy model" ||
		banners[2].CarID != nil || banners[2].BrandID != nil {
		t.Fatalf("expected only the matching legacy slot to link and all three slots to survive: %+v", banners)
	}
	migrate("down")
	var modelName string
	if err := db.Raw("SELECT car_model FROM banners WHERE position = 1").Scan(&modelName).Error; err != nil || modelName != "Legacy model" {
		t.Fatalf("rollback must preserve the legacy content: %q (%v)", modelName, err)
	}
}
