package router

import (
	"encoding/json"
	"testing"

	"catalog/internal/dto"
	"catalog/internal/model"
	"catalog/internal/repository"
)

const fleetPath = "/api/v1/catalog/admin/fleet-summary"

func TestFleetSummaryAccess(t *testing.T) {
	router := newCarTestRouter(t, nil)
	catalogRequest(t, router, "GET", fleetPath, "", "", 401)
	catalogRequest(t, router, "GET", fleetPath, "CUSTOMER", "", 403)
	catalogRequest(t, router, "GET", fleetPath+"?status=active", "ADMIN", "", 400)
}

func TestFleetSummaryPostgres(t *testing.T) {
	db := catalogTestDatabase(t)
	router := newCarTestRouter(t, repository.NewCarRepository(db))
	read := func(token string) dto.FleetSummary {
		t.Helper()
		body := catalogRequest(t, router, "GET", fleetPath, token, "", 200)
		var result struct{ Data dto.FleetSummary }
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatal(err)
		}
		return result.Data
	}
	if empty := read("OWNER"); empty != (dto.FleetSummary{}) {
		t.Fatalf("wrong empty fleet: %+v", empty)
	}
	brand := createCarTestBrand(t, repository.NewBrandRepository(db), "Fleet Test")
	first := createTestCar(t, router, carTestRequest(brand.ID, "First", 400000))
	second := createTestCar(t, router, carTestRequest(brand.ID, "Second", 500000))
	if err := db.Model(&model.Car{}).Where("id = ?", first.ID).Update("status", "active").Error; err != nil {
		t.Fatal(err)
	}
	// Unit status is counted independently of the parent car status.
	for _, unit := range []model.CarUnit{{CarID: first.ID, InternalCode: "COUNT-1", Status: "active"}, {CarID: second.ID, InternalCode: "COUNT-2", Status: "active"}, {CarID: second.ID, InternalCode: "COUNT-3", Status: "inactive"}} {
		if err := db.Create(&unit).Error; err != nil {
			t.Fatal(err)
		}
	}
	want := dto.FleetSummary{Cars: dto.FleetCounts{Total: 2, Active: 1, Inactive: 1}, Units: dto.FleetCounts{Total: 3, Active: 2, Inactive: 1}}
	for _, role := range []string{"ADMIN", "OWNER"} {
		if result := read(role); result != want {
			t.Fatalf("wrong fleet: %+v", result)
		}
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.Close(); err != nil {
		t.Fatal(err)
	}
	catalogRequest(t, router, "GET", fleetPath, "ADMIN", "", 503)
}
