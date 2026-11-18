package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestCustomerSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const origin = "https://web.example.test"
	t.Setenv("WEB_ORIGINS", origin)
	t.Setenv("APP_ENV", "production")
	var revoked sync.Map
	roles := map[string]string{"customer-fixture": "CUSTOMER", "owner-fixture": "OWNER", "admin-fixture": "ADMIN", "staff-fixture": "ADMIN"}
	expiry := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	upstream := testUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "" {
			t.Error("browser cookies must not be forwarded to Identity")
		}
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		switch r.URL.Path {
		case identityPath + "/login":
			if token != "" {
				t.Error("login must ignore supplied Authorization")
			}
			var input struct{ Fixture string }
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Error("login must forward the JSON request")
			}
			_ = json.NewEncoder(w).Encode(gin.H{"data": gin.H{"token": input.Fixture, "expires_at": expiry}})
		case identityPath + "/me":
			_, wasRevoked := revoked.Load(token)
			role, known := roles[token]
			if !known || wasRevoked {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(gin.H{"data": gin.H{"id": "fixture", "role": role, "token": "must-not-leak"}})
		case identityPath + "/customer/change-password":
			if r.Method != http.MethodPost || token != "customer-fixture" {
				t.Error("password change must forward the customer cookie token and POST method")
			}
			var input struct {
				CurrentPassword string `json:"current_password"`
				NewPassword     string `json:"new_password"`
			}
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil ||
				input.CurrentPassword != "fixture-current" || input.NewPassword != "fixture-new" {
				t.Error("password change must forward the Identity request body")
			}
			_, wasRevoked := revoked.Load(token)
			if wasRevoked {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		case identityPath + "/logout":
			revoked.Store(token, true)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected Identity endpoint %s", r.URL.Path)
		}
	})
	app := newRouter(upstream, upstream, upstream, testLogger())
	call := func(method, path, requestOrigin, fixture string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(fixture))
		req.Header.Set("Authorization", "Bearer staff-fixture")
		req.Header.Set("Content-Type", "application/json")
		if requestOrigin != "" {
			req.Header.Set("Origin", requestOrigin)
		}
		if method == http.MethodOptions {
			req.Header.Set("Access-Control-Request-Method", http.MethodPost)
		}
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		res := httptest.NewRecorder()
		app.ServeHTTP(res, req)
		return res
	}
	requireStatus := func(res *httptest.ResponseRecorder, status int) {
		t.Helper()
		if res.Code != status || res.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("status=%d want %d, cache=%q, body=%s", res.Code, status, res.Header().Get("Cache-Control"), res.Body)
		}
	}
	cookieOf := func(res *httptest.ResponseRecorder) *http.Cookie {
		t.Helper()
		cookies := res.Result().Cookies()
		if len(cookies) != 1 {
			t.Fatal("expected one cookie")
		}
		cookie := cookies[0]
		if cookie.Name != customerCookieName || cookie.Path != customerPath || cookie.Domain != "" ||
			!cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
			t.Fatal("invalid customer cookie attributes")
		}
		return cookie
	}
	loginPath := customerPath + "/login"
	mePath := customerPath + "/me"
	logoutPath := customerPath + "/logout"
	changePath := customerPath + "/change-password"
	const changeBody = `{"current_password":"fixture-current","new_password":"fixture-new"}`
	staff := &http.Cookie{Name: staffCookieName, Value: "staff-fixture"}

	preflight := call(http.MethodOptions, loginPath, origin, "")
	requireStatus(preflight, http.StatusNoContent)
	if preflight.Header().Get("Access-Control-Allow-Credentials") != "true" ||
		preflight.Header().Get("Access-Control-Allow-Origin") != origin {
		t.Fatal("allowed origin must support credentials")
	}
	for _, bad := range []string{"", "null", origin + ".invalid"} {
		assertError(t, call(http.MethodPost, loginPath, bad, ""), http.StatusForbidden, "FORBIDDEN_ORIGIN")
		assertError(t, call(http.MethodPost, logoutPath, bad, ""), http.StatusForbidden, "FORBIDDEN_ORIGIN")
		assertError(t, call(http.MethodPost, changePath, bad, changeBody), http.StatusForbidden, "FORBIDDEN_ORIGIN")
	}
	requireStatus(call(http.MethodGet, mePath, origin, "", staff), http.StatusUnauthorized)
	requireStatus(call(http.MethodPost, changePath, origin, changeBody, staff), http.StatusUnauthorized)
	for _, role := range []string{"admin-fixture", "owner-fixture"} {
		res := call(http.MethodPost, loginPath, origin, `{"fixture":"`+role+`"}`)
		assertError(t, res, http.StatusForbidden, "CUSTOMER_REQUIRED")
		if _, ok := revoked.Load(role); !ok || len(res.Result().Cookies()) != 0 {
			t.Fatal("rejecting a staff login must revoke its unused session")
		}
	}
	staffAsCustomer := &http.Cookie{Name: customerCookieName, Value: staff.Value}
	assertError(t, call(http.MethodGet, mePath, origin, "", staffAsCustomer), http.StatusForbidden, "CUSTOMER_REQUIRED")
	assertError(t, call(http.MethodPost, logoutPath, origin, "", staffAsCustomer), http.StatusForbidden, "CUSTOMER_REQUIRED")
	res := call(http.MethodPost, loginPath, origin, `{"fixture":"customer-fixture"}`, staff)
	requireStatus(res, http.StatusOK)
	if strings.Contains(res.Body.String(), "token") {
		t.Fatal("login response must contain only the public profile")
	}
	customer := cookieOf(res)
	if !customer.Expires.Equal(expiry) {
		t.Fatal("cookie expiry must match Identity")
	}
	res = call(http.MethodGet, mePath, origin, "", staff, customer)
	requireStatus(res, http.StatusOK)
	if !strings.Contains(res.Body.String(), `"role":"CUSTOMER"`) || strings.Contains(res.Body.String(), "token") {
		t.Fatal("me must return only the customer profile")
	}
	res = call(http.MethodPost, changePath, origin, changeBody, staff, customer)
	requireStatus(res, http.StatusNoContent)
	if len(res.Result().Cookies()) != 0 || res.Body.Len() != 0 {
		t.Fatal("password change must preserve the session and return an empty response")
	}
	res = call(http.MethodPost, logoutPath, origin, "", staff, customer)
	requireStatus(res, http.StatusNoContent)
	if cookieOf(res).MaxAge >= 0 {
		t.Fatal("logout must delete the customer cookie")
	}
	requireStatus(call(http.MethodGet, mePath, origin, "", customer), http.StatusUnauthorized)
	requireStatus(call(http.MethodPost, changePath, origin, changeBody, customer), http.StatusUnauthorized)
	requireStatus(call(http.MethodGet, staffPath+"/me", origin, "", staff), http.StatusOK)
	res = call(http.MethodPost, logoutPath, origin, "", customer)
	requireStatus(res, http.StatusNoContent)
	if cookieOf(res).MaxAge >= 0 {
		t.Fatal("expired sessions must also clear the customer cookie")
	}
}
