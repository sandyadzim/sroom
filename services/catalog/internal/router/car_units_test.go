package router

import (
	"strings"
	"testing"

	"catalog/internal/dto"
	"catalog/internal/handler"
	"catalog/internal/model"
	"catalog/internal/repository"
	"catalog/internal/service"
)

const adminUnitsPath = "/api/v1/catalog/admin/units"

func TestCarUnitAuthorizationAndInput(t *testing.T) {
	units := handler.NewCarUnitHandler(service.NewCarUnitService(nil))
	router := newCatalogTestRouter(t, nil, nil, nil, nil, units)
	listPath := adminCarsPath + "/" + missingBrandID + "/units"
	detailPath := adminUnitsPath + "/" + missingBrandID
	for _, route := range []struct{ method, path string }{
		{"GET", listPath}, {"POST", listPath}, {"GET", detailPath}, {"PATCH", detailPath},
	} {
		catalogRequest(t, router, route.method, route.path, "", "", 401)
		catalogRequest(t, router, route.method, route.path, "CUSTOMER", "", 403)
	}
	for _, path := range []string{adminCarsPath + "/bad/units", adminUnitsPath + "/bad"} {
		catalogRequest(t, router, "GET", path, "ADMIN", "", 400)
	}
	for _, query := range []string{"?page=0", "?limit=101", "?status=reserved", "?brand_id=unknown"} {
		catalogRequest(t, router, "GET", listPath+query, "ADMIN", "", 400)
	}
	for _, body := range []string{
		`{"status":"active"}`, `{"internal_code":"  "}`, `{"internal_code":"UNIT\u0000"}`,
		`{"internal_code":"UNIT","status":""}`, `{"internal_code":"UNIT","status":null}`,
		`{"internal_code":"UNIT","car_id":"other"}`,
		carJSON(t, dto.CreateCarUnitRequest{InternalCode: strings.Repeat("a", 101)}),
	} {
		catalogRequest(t, router, "POST", listPath, "OWNER", body, 400)
	}
	for _, body := range []string{`{}`, `{"internal_code":null}`, `{"status":"reserved"}`, `{"car_id":"other"}`} {
		catalogRequest(t, router, "PATCH", detailPath, "OWNER", body, 400)
	}
}

func TestCarUnitsPostgres(t *testing.T) {
	db := catalogTestDatabase(t)
	brand := createCarTestBrand(t, repository.NewBrandRepository(db), "Toyota")
	cars := handler.NewCarHandler(service.NewCarService(repository.NewCarRepository(db, false), carTestImages{}, ""))
	units := handler.NewCarUnitHandler(service.NewCarUnitService(repository.NewCarUnitRepository(db, false)))
	router := newCatalogTestRouter(t, nil, nil, cars, nil, units)
	active := model.CarActive
	input := carTestRequest(brand.ID, "Innova", 600000)
	input.Status = &active
	variant := createTestCar(t, router, input)
	other := createTestCar(t, router, carTestRequest(brand.ID, "Rush", 500000))
	listPath := adminCarsPath + "/" + variant.ID + "/units"
	otherPath := adminCarsPath + "/" + other.ID + "/units"
	missingCarPath := adminCarsPath + "/" + missingBrandID + "/units"
	missingUnitPath := adminUnitsPath + "/" + missingBrandID

	empty := catalogRequest(t, router, "GET", listPath, "OWNER", "", 200)
	if string(empty) != `{"data":[],"page":{"number":1,"limit":20,"total":0}}` {
		t.Fatalf("unexpected empty unit list: %s", empty)
	}
	catalogRequest(t, router, "GET", missingCarPath, "OWNER", "", 404)
	catalogRequest(t, router, "POST", missingCarPath, "ADMIN", `{"internal_code":"MISSING-001"}`, 404)
	catalogRequest(t, router, "GET", missingUnitPath, "ADMIN", "", 404)
	catalogRequest(t, router, "PATCH", missingUnitPath, "ADMIN", `{"status":"active"}`, 404)

	var result struct {
		Message string
		Data    dto.CarUnitResponse
	}
	decodeCarJSON(t, catalogRequest(t, router, "POST", listPath, "ADMIN", `{"internal_code":"  INNOVA-001 "}`, 201), &result)
	first := result.Data
	if first.ID == "" || first.CarID != variant.ID || first.InternalCode != "INNOVA-001" ||
		first.Status != model.UnitInactive || first.CreatedAt == "" || first.UpdatedAt == "" || result.Message == "" {
		t.Fatalf("unexpected created unit: %+v", result)
	}
	decodeCarJSON(t, catalogRequest(t, router, "POST", listPath, "OWNER", `{"internal_code":"INNOVA-002","status":"active"}`, 201), &result)
	second := result.Data
	catalogRequest(t, router, "POST", otherPath, "OWNER", `{"internal_code":"innova-001"}`, 409)
	catalogRequest(t, router, "POST", otherPath, "ADMIN", `{"internal_code":"RUSH-001"}`, 201)

	var page dto.CarUnitListResponse
	decodeCarJSON(t, catalogRequest(t, router, "GET", listPath+"?limit=1", "ADMIN", "", 200), &page)
	if len(page.Data) != 1 || page.Data[0].ID != second.ID || page.Page != (dto.CarPage{Number: 1, Limit: 1, Total: 2}) {
		t.Fatalf("unexpected first page: %+v", page)
	}
	decodeCarJSON(t, catalogRequest(t, router, "GET", listPath+"?page=2&limit=1", "ADMIN", "", 200), &page)
	if len(page.Data) != 1 || page.Data[0].ID != first.ID || page.Page.Number != 2 {
		t.Fatalf("unexpected second page: %+v", page)
	}
	decodeCarJSON(t, catalogRequest(t, router, "GET", listPath+"?status=active", "ADMIN", "", 200), &page)
	if len(page.Data) != 1 || page.Data[0].ID != second.ID || page.Page.Total != 1 {
		t.Fatalf("unexpected status filter: %+v", page)
	}

	detailPath := adminUnitsPath + "/" + second.ID
	catalogRequest(t, router, "PATCH", detailPath, "ADMIN", `{"internal_code":"innova-001","status":"inactive"}`, 409)
	decodeCarJSON(t, catalogRequest(t, router, "GET", detailPath, "OWNER", "", 200), &result)
	if result.Data.Status != model.UnitActive || result.Data.InternalCode != second.InternalCode {
		t.Fatal("conflicting update must leave the whole unit unchanged")
	}
	decodeCarJSON(t, catalogRequest(t, router, "PATCH", detailPath, "OWNER", `{"internal_code":" INNOVA-003 ","status":"inactive"}`, 200), &result)
	if result.Data.ID != second.ID || result.Data.CarID != variant.ID || result.Data.InternalCode != "INNOVA-003" ||
		result.Data.Status != model.UnitInactive || result.Data.CreatedAt != second.CreatedAt {
		t.Fatalf("unexpected update: %+v", result.Data)
	}
	catalogRequest(t, router, "DELETE", detailPath, "OWNER", "", 404)
	catalogRequest(t, router, "GET", detailPath, "OWNER", "", 200)

	var public dto.CarList
	body := catalogRequest(t, router, "GET", publicCarsPath, "", "", 200)
	decodeCarJSON(t, body, &public)
	if len(public.Data) != 1 || public.Data[0].ID != variant.ID || strings.Contains(string(body), first.ID) ||
		strings.Contains(string(body), "internal_code") {
		t.Fatalf("public cars should contain only the active variant and no unit metadata: %s", body)
	}
	body = catalogRequest(t, router, "GET", publicCarsPath+"/"+variant.Slug, "", "", 200)
	if strings.Contains(string(body), first.ID) || strings.Contains(string(body), "internal_code") {
		t.Fatal("public car detail should not expose unit metadata")
	}
}
