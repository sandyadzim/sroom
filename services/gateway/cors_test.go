package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestStaffBrowserCORS(t *testing.T) {
	t.Setenv("WEB_ORIGINS", "http://localhost:5173, https://staff.example.test")
	gin.SetMode(gin.TestMode)
	upstream := testUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("expected an authenticated GET and no upstream preflight")
		}
		w.WriteHeader(http.StatusForbidden)
	})
	app := newRouter(upstream, upstream, testLogger())
	path := "/api/v1/identity/admins"
	origin := "https://staff.example.test"

	preflight := httptest.NewRequest(http.MethodOptions, path, nil)
	preflight.Header.Set("Origin", origin)
	preflight.Header.Set("Access-Control-Request-Method", "GET")
	preflight.Header.Set("Access-Control-Request-Headers", "authorization")
	response := request(app, preflight)
	if response.Code != http.StatusNoContent ||
		response.Header().Get("Access-Control-Allow-Origin") != origin ||
		!strings.Contains(response.Header().Get("Access-Control-Allow-Headers"), "Authorization") {
		t.Fatal("expected allowed staff preflight")
	}

	get := httptest.NewRequest(http.MethodGet, path, nil)
	get.Header.Set("Origin", origin)
	get.Header.Set("Authorization", "Bearer test-token")
	response = request(app, get)
	if response.Code != http.StatusForbidden || response.Header().Get("Access-Control-Allow-Origin") != origin {
		t.Fatal("expected browser to receive the upstream authorization error")
	}

	preflight = httptest.NewRequest(http.MethodOptions, path, nil)
	preflight.Header.Set("Origin", "https://staff.example.test.evil.test")
	response = request(app, preflight)
	if response.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("unexpected CORS access for an unlisted origin")
	}
}
