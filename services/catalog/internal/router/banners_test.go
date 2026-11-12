package router

import (
	"context"
	"encoding/json"
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
	router := newCatalogTestRouter(t, handler.NewBrandHandler(nil), banners, handler.NewCarHandler(nil))
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
	router := newCatalogTestRouter(t, handler.NewBrandHandler(nil), banners, handler.NewCarHandler(nil))

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
