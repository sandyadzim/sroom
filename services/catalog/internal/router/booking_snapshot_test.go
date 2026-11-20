package router

import (
	"net/http"
	"slices"
	"testing"

	"catalog/internal/dto"
	"catalog/internal/handler"
	"catalog/internal/model"
	"catalog/internal/repository"
	"catalog/internal/service"
)

const testBookingToken = "booking-test-token"

func snapshotPath(id string) string {
	return "/internal/v1/variants/" + id + "/booking-snapshot"
}

func TestBookingSnapshotAccess(t *testing.T) {
	cars := handler.NewCarHandler(service.NewCarService(nil, nil, "", nil))
	enabled := newCatalogTestRouterWithBookingToken(t, nil, nil, cars, nil, nil, testBookingToken)
	catalogRequest(t, enabled, http.MethodGet, snapshotPath(missingBrandID), "", "", 401)
	catalogRequest(t, enabled, http.MethodGet, snapshotPath(missingBrandID), "ADMIN", "", 401)
	catalogRequest(t, enabled, http.MethodGet, snapshotPath("invalid"), testBookingToken, "", 400)

	disabled := newCatalogTestRouter(t, nil, nil, cars, nil, nil)
	catalogRequest(t, disabled, http.MethodGet, snapshotPath(missingBrandID), testBookingToken, "", 404)
}

func TestBookingSnapshotPostgres(t *testing.T) {
	db := catalogTestDatabase(t)
	brand := createCarTestBrand(t, repository.NewBrandRepository(db), "Toyota")
	cars := handler.NewCarHandler(service.NewCarService(repository.NewCarRepository(db), carTestImages{}, "", nil))
	units := handler.NewCarUnitHandler(service.NewCarUnitService(repository.NewCarUnitRepository(db)))
	router := newCatalogTestRouterWithBookingToken(t, nil, nil, cars, nil, units, testBookingToken)
	active := model.CarActive
	input := carTestRequest(brand.ID, "Innova", 600000)
	input.Status = &active
	variant := createTestCar(t, router, input)
	unitPath := adminCarsPath + "/" + variant.ID + "/units"

	readSnapshot := func() dto.BookingSnapshot {
		t.Helper()
		var result struct {
			Data dto.BookingSnapshot
		}
		decodeCarJSON(t, catalogRequest(t, router, http.MethodGet, snapshotPath(variant.ID), testBookingToken, "", 200), &result)
		return result.Data
	}
	snapshot := readSnapshot()
	if snapshot.VariantID != variant.ID || snapshot.VehicleLabel != "Toyota Innova "+input.Trim ||
		snapshot.DailyRateIDR != 600000 || snapshot.Status != model.CarActive ||
		snapshot.EligibleUnitIDs == nil || len(snapshot.EligibleUnitIDs) != 0 {
		t.Fatalf("unexpected snapshot without units: %+v", snapshot)
	}

	var unit struct {
		Data dto.CarUnitResponse
	}
	decodeCarJSON(t, catalogRequest(t, router, "POST", unitPath, "ADMIN", `{"internal_code":"ACTIVE","status":"active"}`, 201), &unit)
	catalogRequest(t, router, "POST", unitPath, "ADMIN", `{"internal_code":"INACTIVE","status":"inactive"}`, 201)
	if got := readSnapshot().EligibleUnitIDs; !slices.Equal(got, []string{unit.Data.ID}) {
		t.Fatalf("eligible units = %v, want only the active unit", got)
	}

	catalogRequest(t, router, "PATCH", adminCarsPath+"/"+variant.ID, "ADMIN", `{"status":"inactive"}`, 200)
	snapshot = readSnapshot()
	if snapshot.Status != model.CarInactive || snapshot.EligibleUnitIDs == nil || len(snapshot.EligibleUnitIDs) != 0 {
		t.Fatalf("unexpected inactive snapshot: %+v", snapshot)
	}
	catalogRequest(t, router, http.MethodGet, snapshotPath(missingBrandID), testBookingToken, "", 404)
}

func TestDeactivationChangesBookingSnapshotPostgres(t *testing.T) {
	db := catalogTestDatabase(t)
	brand := createCarTestBrand(t, repository.NewBrandRepository(db), "Booking Brand")
	cars := handler.NewCarHandler(service.NewCarService(repository.NewCarRepository(db), carTestImages{}, "", nil))
	units := handler.NewCarUnitHandler(service.NewCarUnitService(repository.NewCarUnitRepository(db)))
	router := newCatalogTestRouterWithBookingToken(t, nil, nil, cars, nil, units, testBookingToken)
	active := model.CarActive
	input := carTestRequest(brand.ID, "Innova", 450000)
	input.Status = &active
	variant := createTestCar(t, router, input)
	carPath := adminCarsPath + "/" + variant.ID
	var unit struct{ Data dto.CarUnitResponse }
	decodeCarJSON(t, catalogRequest(t, router, "POST", carPath+"/units", "ADMIN", `{"internal_code":"BOOKING-001","status":"active"}`, 201), &unit)
	unitPath := adminUnitsPath + "/" + unit.Data.ID

	assertSnapshot := func(status string, unitIDs []string) {
		t.Helper()
		response := catalogRequest(t, router, "GET", snapshotPath(variant.ID), testBookingToken, "", 200)
		var snapshot struct{ Data dto.BookingSnapshot }
		decodeCarJSON(t, response, &snapshot)
		if snapshot.Data.Status != status || !slices.Equal(snapshot.Data.EligibleUnitIDs, unitIDs) {
			t.Fatalf("unexpected booking snapshot: %+v", snapshot.Data)
		}
	}

	assertSnapshot(model.CarActive, []string{unit.Data.ID})
	catalogRequest(t, router, "PATCH", unitPath, "ADMIN", `{"status":"inactive"}`, 200)
	assertSnapshot(model.CarActive, []string{})
	catalogRequest(t, router, "PATCH", unitPath, "ADMIN", `{"status":"active"}`, 200)
	assertSnapshot(model.CarActive, []string{unit.Data.ID})

	catalogRequest(t, router, "PATCH", carPath, "OWNER", `{"status":"inactive"}`, 200)
	assertSnapshot(model.CarInactive, []string{})
	var savedUnit struct{ Data dto.CarUnitResponse }
	decodeCarJSON(t, catalogRequest(t, router, "GET", unitPath, "ADMIN", "", 200), &savedUnit)
	if savedUnit.Data.Status != model.UnitActive {
		t.Fatal("deactivating a car must not change its unit status")
	}
	catalogRequest(t, router, "PATCH", carPath, "OWNER", `{"status":"active"}`, 200)
	assertSnapshot(model.CarActive, []string{unit.Data.ID})
}
