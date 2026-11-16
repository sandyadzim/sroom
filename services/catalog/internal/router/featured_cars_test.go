package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"catalog/internal/dto"
	"catalog/internal/handler"
	"catalog/internal/model"
	"catalog/internal/repository"
	"catalog/internal/service"
)

const adminFeaturedPath = "/api/v1/catalog/admin/featured-cars"
const publicFeaturedPath = "/api/v1/catalog/home/featured-cars"

func TestFeaturedCarsAuthorizationAndInput(t *testing.T) {
	featured := handler.NewFeaturedCarHandler(service.NewFeaturedCarService(nil, ""))
	router := newCatalogTestRouter(t, nil, nil, nil, featured, nil)
	for _, method := range []string{"GET", "PUT"} {
		catalogRequest(t, router, method, adminFeaturedPath, "", "", 401)
		catalogRequest(t, router, method, adminFeaturedPath, "CUSTOMER", "", 403)
	}
	for _, body := range []string{
		`{}`,
		`{"car_ids":null}`,
		`{"car_ids":[null]}`,
		`{"car_ids":"invalid"}`,
		`{"car_ids":["invalid"]}`,
		`{"car_ids":[],"status":"active"}`,
	} {
		catalogRequest(t, router, "PUT", adminFeaturedPath, "ADMIN", body, 400)
	}
}

func TestFeaturedCarsPostgres(t *testing.T) {
	db := catalogTestDatabase(t)
	brand := createCarTestBrand(t, repository.NewBrandRepository(db), "Toyota")
	cars := handler.NewCarHandler(service.NewCarService(repository.NewCarRepository(db), carTestImages{}, ""))
	featured := handler.NewFeaturedCarHandler(service.NewFeaturedCarService(repository.NewFeaturedCarRepository(db), "https://assets.example.test/"))
	router := newCatalogTestRouter(t, nil, nil, cars, featured, nil)

	if body := catalogRequest(t, router, "GET", publicFeaturedPath, "", "", 200); string(body) != `{"data":[]}` {
		t.Fatalf("expected an empty public list: %s", body)
	}

	active := model.CarActive
	var variants []dto.AdminCarDetail
	for _, name := range []string{"Fortuner", "Yaris", "Rush", "Avanza"} {
		input := carTestRequest(brand.ID, name, 400000)
		input.Status = &active
		variants = append(variants, createTestCar(t, router, input))
	}
	first, second, third, fourth := variants[0].ID, variants[1].ID, variants[2].ID, variants[3].ID

	payload := func(ids ...string) string {
		return carJSON(t, dto.UpdateFeaturedCarsRequest{CarIDs: ids})
	}
	var saved struct {
		Data    []dto.AdminFeaturedCar
		Message string
	}
	decodeCarJSON(t, catalogRequest(t, router, "PUT", adminFeaturedPath, "OWNER", payload(third, first, second), 200), &saved)
	if saved.Message == "" || len(saved.Data) != 3 ||
		saved.Data[0].ID != third || saved.Data[1].ID != first || saved.Data[2].ID != second {
		t.Fatalf("expected the submitted order in the admin response: %+v", saved)
	}
	var public struct{ Data []dto.CarCard }
	decodeCarJSON(t, catalogRequest(t, router, "GET", publicFeaturedPath, "", "", 200), &public)
	if len(public.Data) != 3 || public.Data[0].ID != third || public.Data[1].ID != first || public.Data[2].ID != second ||
		public.Data[0].Cover == nil || public.Data[0].Cover.URL == nil || *public.Data[0].Cover.URL != "https://assets.example.test/"+uploadedKey {
		t.Fatalf("expected only selected cars in order with Catalog cover data: %+v", public.Data)
	}

	catalogRequest(t, router, "PUT", adminFeaturedPath, "ADMIN", payload(first, strings.ToUpper(first)), 400)
	catalogRequest(t, router, "PUT", adminFeaturedPath, "ADMIN", payload(first, second, third, fourth), 400)
	catalogRequest(t, router, "PUT", adminFeaturedPath, "ADMIN", payload(first, missingBrandID), 404)
	decodeCarJSON(t, catalogRequest(t, router, "GET", adminFeaturedPath, "ADMIN", "", 200), &saved)
	if len(saved.Data) != 3 || saved.Data[0].ID != third || saved.Data[1].ID != first || saved.Data[2].ID != second {
		t.Fatalf("invalid updates must preserve the whole selection: %+v", saved.Data)
	}

	catalogRequest(t, router, "PUT", adminFeaturedPath, "ADMIN", payload(first, second, third), 200)
	catalogRequest(t, router, "PATCH", adminCarsPath+"/"+second, "ADMIN", `{"status":"inactive"}`, 200)
	decodeCarJSON(t, catalogRequest(t, router, "GET", publicFeaturedPath, "", "", 200), &public)
	if len(public.Data) != 2 || public.Data[0].ID != first || public.Data[1].ID != third {
		t.Fatalf("the public list should skip inactive cars and keep the rest in order: %+v", public.Data)
	}
	decodeCarJSON(t, catalogRequest(t, router, "GET", adminFeaturedPath, "ADMIN", "", 200), &saved)
	if len(saved.Data) != 3 || saved.Data[1].ID != second || saved.Data[1].Status != model.CarInactive {
		t.Fatalf("the admin list should retain the selection and show its status: %+v", saved.Data)
	}
	catalogRequest(t, router, "PATCH", adminCarsPath+"/"+second, "ADMIN", `{"status":"active"}`, 200)
	decodeCarJSON(t, catalogRequest(t, router, "GET", publicFeaturedPath, "", "", 200), &public)
	if len(public.Data) != 3 || public.Data[1].ID != second {
		t.Fatalf("reactivation should restore the car in its saved position: %+v", public.Data)
	}

	// Replacements starting from an empty table must each succeed as a complete list.
	catalogRequest(t, router, "PUT", adminFeaturedPath, "ADMIN", `{"car_ids":[]}`, 200)
	bodies := []string{payload(first, second), payload(third, fourth)}
	var writes sync.WaitGroup
	for _, body := range bodies {
		writes.Go(func() {
			req := httptest.NewRequest("PUT", adminFeaturedPath, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer ADMIN")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Errorf("concurrent replacement failed: %d %s", rec.Code, rec.Body)
			}
		})
	}
	writes.Wait()
	decodeCarJSON(t, catalogRequest(t, router, "GET", publicFeaturedPath, "", "", 200), &public)
	if len(public.Data) != 2 || !((public.Data[0].ID == first && public.Data[1].ID == second) ||
		(public.Data[0].ID == third && public.Data[1].ID == fourth)) {
		t.Fatalf("concurrent saves must leave one complete selection: %+v", public.Data)
	}

	catalogRequest(t, router, "PUT", adminFeaturedPath, "ADMIN", `{"car_ids":[]}`, 200)
	if body := catalogRequest(t, router, "GET", publicFeaturedPath, "", "", 200); string(body) != `{"data":[]}` {
		t.Fatalf("expected an empty selection after clearing: %s", body)
	}
}
