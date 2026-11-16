package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestStaffSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("WEB_ORIGINS", "https://staff.example.test")
	t.Setenv("APP_ENV", "production")
	revoked := false
	upstream := testUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "" {
			t.Error("staff cookie must not be forwarded upstream")
		}
		switch r.URL.Path {
		case identityPath + "/login":
			if r.Header.Get("Authorization") != "" {
				t.Error("login must not forward an existing credential")
			}
			_ = json.NewEncoder(w).Encode(gin.H{"data": gin.H{"token": "fixture-token", "expires_at": time.Now().Add(time.Hour)}})
		case identityPath + "/me", identityPath + "/admins/fixture-id", "/api/v1/catalog/admin/brands":
			if r.Header.Get("Authorization") != "Bearer fixture-token" {
				t.Error("expected the session translated to a Bearer token")
			}
			if strings.HasSuffix(r.URL.Path, "fixture-id") && r.URL.Query().Get("test") != "A+B" {
				t.Error("proxy changed the query")
			}
			if revoked {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_, _ = io.WriteString(w, `{"data":{"id":"fixture-user","role":"OWNER","token":"unexpected-upstream-field"}}`)
		case identityPath + "/logout":
			if r.Header.Get("Authorization") != "Bearer fixture-token" {
				t.Error("logout must revoke the current session")
			}
			revoked = true
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected upstream path %s", r.URL.Path)
		}
	})
	app := newRouter(upstream, upstream, testLogger())
	const origin = "https://staff.example.test"
	call := func(method, path string, cookie *http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Origin", origin)
		if cookie != nil {
			req.AddCookie(cookie)
			req.Header.Set("Authorization", "Bearer ignored-token")
		}
		return request(app, req)
	}

	preflight := httptest.NewRequest(http.MethodOptions, staffPath+"/login", nil)
	preflight.Header.Set("Origin", origin)
	preflight.Header.Set("Access-Control-Request-Method", "POST")
	res := request(app, preflight)
	if res.Code != http.StatusNoContent || res.Header().Get("Access-Control-Allow-Credentials") != "true" ||
		res.Header().Get("Access-Control-Allow-Origin") != origin {
		t.Fatal("expected credentialed CORS preflight")
	}

	if res := call(http.MethodGet, staffPath+"/me", nil); res.Code != http.StatusUnauthorized {
		t.Fatal("missing staff cookie must return 401")
	}
	res = call(http.MethodPost, staffPath+"/login", nil)
	cookies := res.Result().Cookies()
	if res.Code != http.StatusOK || res.Header().Get("Cache-Control") != "no-store" ||
		!strings.Contains(res.Body.String(), "OWNER") || strings.Contains(res.Body.String(), "token") ||
		len(cookies) != 1 {
		t.Fatal("staff login must return only the profile and set a cookie")
	}
	cookie := cookies[0]
	if cookie.Name != staffCookieName || cookie.Value != "fixture-token" || !cookie.HttpOnly ||
		!cookie.Secure || cookie.Path != staffPath || cookie.SameSite != http.SameSiteLaxMode ||
		cookie.Domain != "" || !cookie.Expires.After(time.Now()) {
		t.Fatal("incorrect staff session cookie")
	}
	for _, path := range []string{"/me", "/admins/fixture-id?test=A%2BB", "/catalog/admin/brands"} {
		if res := call(http.MethodGet, staffPath+path, cookie); res.Code != http.StatusOK {
			t.Errorf("%s: expected successful authenticated request", path)
		}
	}
	for _, otherOrigin := range []string{"", "null", origin + ".invalid"} {
		req := httptest.NewRequest(http.MethodPost, staffPath+"/logout", nil)
		req.AddCookie(cookie)
		if otherOrigin != "" {
			req.Header.Set("Origin", otherOrigin)
		}
		if res := request(app, req); res.Code != http.StatusForbidden || revoked {
			t.Fatal("untrusted mutation must be rejected before logout")
		}
	}

	res = call(http.MethodPost, staffPath+"/logout", cookie)
	cookies = res.Result().Cookies()
	if res.Code != http.StatusNoContent || !revoked || len(cookies) != 1 ||
		cookies[0].Name != staffCookieName || cookies[0].Path != staffPath || cookies[0].MaxAge != -1 {
		t.Fatal("logout must revoke the session and expire the cookie")
	}
	if res := call(http.MethodGet, staffPath+"/me", cookie); res.Code != http.StatusUnauthorized {
		t.Fatal("revoked session must return 401")
	}
	res = call(http.MethodPost, identityPath+"/login", nil)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "fixture-token") ||
		res.Header().Get("Set-Cookie") != "" || res.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Fatal("public login contract must still return the token")
	}
}

func TestStaffLoginRejectsCustomer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("WEB_ORIGINS", "https://staff.example.test")
	revoked := false
	upstream := testUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case identityPath + "/login":
			_ = json.NewEncoder(w).Encode(gin.H{"data": gin.H{"token": "fixture-customer", "expires_at": time.Now().Add(time.Hour)}})
		case identityPath + "/me":
			_, _ = io.WriteString(w, `{"data":{"id":"fixture-user","role":"CUSTOMER"}}`)
		case identityPath + "/logout":
			revoked = r.Header.Get("Authorization") == "Bearer fixture-customer"
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Error("unexpected upstream request")
		}
	})
	app := newRouter(upstream, upstream, testLogger())
	req := httptest.NewRequest(http.MethodPost, staffPath+"/login", nil)
	req.Header.Set("Origin", "https://staff.example.test")
	res := request(app, req)
	assertError(t, res, http.StatusForbidden, "STAFF_REQUIRED")
	if !revoked || res.Header().Get("Set-Cookie") != "" || strings.Contains(res.Body.String(), "fixture-customer") {
		t.Fatal("customer must receive no staff session")
	}
}

func TestStaffCustomerStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const origin = "https://staff.example.test"
	t.Setenv("WEB_ORIGINS", origin)
	calls := 0
	upstream := testUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		body, err := io.ReadAll(r.Body)
		if err != nil || string(body) != `{"status":"inactive"}` ||
			r.Method != http.MethodPatch || r.URL.Path != identityPath+"/customers/test-id/status" ||
			r.Header.Get("Authorization") != "Bearer fixture-session" {
			t.Error("expected customer status request with the staff session forwarded to Identity")
		}
		// The gateway must preserve Identity's role decision.
		w.WriteHeader(http.StatusForbidden)
	})
	app := newRouter(upstream, upstream, testLogger())

	for _, test := range []struct {
		name, origin string
		cookie       bool
		status       int
	}{
		{"missing session", origin, false, http.StatusUnauthorized},
		{"missing origin", "", true, http.StatusForbidden},
		{"forwarded to Identity", origin, true, http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPatch, staffPath+"/customers/test-id/status", strings.NewReader(`{"status":"inactive"}`))
			req.Header.Set("Content-Type", "application/json")
			if test.origin != "" {
				req.Header.Set("Origin", test.origin)
			}
			if test.cookie {
				req.AddCookie(&http.Cookie{Name: staffCookieName, Value: "fixture-session"})
			}
			if res := request(app, req); res.Code != test.status {
				t.Fatalf("status = %d, want %d", res.Code, test.status)
			}
		})
	}
	if calls != 1 {
		t.Fatalf("upstream calls = %d, want 1", calls)
	}
}

func TestStaffFeaturedCarsRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const origin = "https://staff.example.test"
	const payload = `{"car_ids":[]}`
	t.Setenv("WEB_ORIGINS", origin)
	upstream := testUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/catalog/admin/featured-cars" || r.Header.Get("Authorization") != "Bearer fixture-session" {
			t.Error("expected Featured Cars path and translated staff session")
		}
		if r.Method == http.MethodPut {
			body, err := io.ReadAll(r.Body)
			if err != nil || string(body) != payload {
				t.Error("expected the Featured Cars selection payload")
			}
		}
		w.WriteHeader(http.StatusOK)
	})
	router := newRouter(upstream, upstream, testLogger())
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		req := httptest.NewRequest(method, staffPath+"/catalog/admin/featured-cars", strings.NewReader(payload))
		req.Header.Set("Origin", origin)
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: staffCookieName, Value: "fixture-session"})
		if res := request(router, req); res.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, want 200", method, res.Code)
		}
	}
}

func TestStaffCarUnitRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const origin = "https://staff.example.test"
	t.Setenv("WEB_ORIGINS", origin)
	upstream := testUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer fixture-session" {
			t.Error("expected translated staff session")
		}
		_, _ = io.WriteString(w, r.Method+" "+r.URL.RequestURI())
	})
	app := newRouter(upstream, upstream, testLogger())
	for _, route := range []struct{ method, path string }{
		{"GET", "/catalog/admin/cars/test-id/units?status=active"},
		{"POST", "/catalog/admin/cars/test-id/units"},
		{"GET", "/catalog/admin/units/test-id"},
		{"PATCH", "/catalog/admin/units/test-id"},
	} {
		req := httptest.NewRequest(route.method, staffPath+route.path, nil)
		req.Header.Set("Origin", origin)
		req.AddCookie(&http.Cookie{Name: staffCookieName, Value: "fixture-session"})
		res := request(app, req)
		if res.Code != http.StatusOK || res.Body.String() != route.method+" /api/v1"+route.path {
			t.Fatalf("%s %s: unexpected proxy response %d %s", route.method, route.path, res.Code, res.Body)
		}
	}
}
