package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const overviewFixture = `{"data":{"period":{"from":"2026-11-14","to":"2026-11-20"},"timezone":"Asia/Makassar","currency":"IDR","simulated":true,"summary":{"total_bookings":2},"daily":[],"recent_bookings":[]}}`
const fleetFixture = `{"data":{"cars":{"total":2,"active":1,"inactive":1},"units":{"total":3,"active":2,"inactive":1}}}`

func TestStaffReportsAndOverview(t *testing.T) {
	bookingCalls, catalogCalls := 0, 0
	booking := testUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		bookingCalls++
		if r.Header.Get("Authorization") != "Bearer staff-fixture" || r.Header.Get("Cookie") != "" || r.Header.Get("X-Request-ID") != "report-test" {
			t.Error("wrong booking auth/headers")
		}
		switch r.URL.Path {
		case "/api/v1/booking/admin/overview":
			if r.URL.RawQuery != "" {
				t.Error("unexpected overview query")
			}
			fmt.Fprint(w, overviewFixture)
		case "/api/v1/booking/admin/reports":
			if r.URL.RawQuery != "from=2026-11-14&to=2026-11-20" {
				t.Error("report query lost")
			}
			fmt.Fprint(w, `{"data":{"summary":{"total_bookings":2}}}`)
		default:
			t.Error("wrong booking path", r.URL.Path)
			w.WriteHeader(404)
		}
	})
	catalog := testUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		catalogCalls++
		if r.URL.Path != "/api/v1/catalog/admin/fleet-summary" || r.Header.Get("Authorization") != "Bearer staff-fixture" || r.Header.Get("Cookie") != "" || r.Header.Get("X-Request-ID") != "report-test" {
			t.Error("wrong catalog auth/headers")
		}
		fmt.Fprint(w, fleetFixture)
	})
	router := newRouter(booking, catalog, booking, booking, testLogger())
	for _, path := range []string{"/api/v1/staff/overview", "/api/v1/staff/reports?from=2026-11-14&to=2026-11-20"} {
		assertError(t, request(router, httptest.NewRequest("GET", path, nil)), 401, "UNAUTHENTICATED")
		req := httptest.NewRequest("GET", path, nil)
		req.AddCookie(&http.Cookie{Name: staffCookieName, Value: "staff-fixture"})
		req.Header.Set("Authorization", "Bearer ignored")
		req.Header.Set("X-Request-ID", "report-test")
		res := request(router, req)
		if res.Code != 200 || res.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s: %d %s", path, res.Code, res.Body.String())
		}
		if strings.Contains(path, "overview") {
			var result struct {
				Data struct {
					Fleet struct {
						Cars  struct{ Total int }
						Units struct{ Total int }
					}
					RecentBookings []any `json:"recent_bookings"`
				}
			}
			if err := json.Unmarshal(res.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Data.Fleet.Cars.Total != 2 || result.Data.Fleet.Units.Total != 3 || result.Data.RecentBookings == nil {
				t.Fatalf("wrong composition: %s", res.Body.String())
			}
		}
	}
	if bookingCalls != 2 || catalogCalls != 1 {
		t.Fatalf("unexpected upstream calls: booking=%d catalog=%d", bookingCalls, catalogCalls)
	}
}

func TestOverviewDependencyErrors(t *testing.T) {
	for _, test := range []struct {
		name, booking, catalog, code         string
		bookingStatus, catalogStatus, status int
	}{
		{"booking unavailable", `{"error":{"code":"DEPENDENCY_UNAVAILABLE","message":"could not read bookings"}}`, fleetFixture, "DEPENDENCY_UNAVAILABLE", 503, 200, 503},
		{"customer denied", `{"error":{"code":"FORBIDDEN","message":"staff access required"}}`, fleetFixture, "FORBIDDEN", 403, 200, 403},
		{"bad booking", `{"data":{}}`, fleetFixture, "INVALID_UPSTREAM_RESPONSE", 200, 200, 502},
		{"catalog unavailable", overviewFixture, `{"error":{"code":"DEPENDENCY_UNAVAILABLE","message":"could not read fleet"}}`, "DEPENDENCY_UNAVAILABLE", 200, 503, 503},
		{"bad fleet", overviewFixture, `{"data":{"cars":{"total":0}}}`, "INVALID_UPSTREAM_RESPONSE", 200, 200, 502},
		{"invalid json", overviewFixture, `oops`, "INVALID_UPSTREAM_RESPONSE", 200, 200, 502},
		{"redirect", overviewFixture, ``, "INVALID_UPSTREAM_RESPONSE", 200, 302, 502},
	} {
		t.Run(test.name, func(t *testing.T) {
			booking := testUpstream(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.bookingStatus)
				fmt.Fprint(w, test.booking)
			})
			catalog := testUpstream(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.catalogStatus)
				fmt.Fprint(w, test.catalog)
			})
			router := newRouter(booking, catalog, booking, booking, testLogger())
			req := httptest.NewRequest("GET", "/api/v1/staff/overview", nil)
			req.AddCookie(&http.Cookie{Name: staffCookieName, Value: "staff-fixture"})
			assertError(t, request(router, req), test.status, test.code)
		})
	}
}
