package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	identity := testUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test-Ignored", "private header")
		_, _ = io.WriteString(w, "identity "+r.Method+" "+r.URL.RequestURI())
	})
	catalog := testUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "catalog "+r.Method+" "+r.URL.RequestURI())
	})
	app := newRouter(identity, catalog, testLogger())

	for _, test := range []struct{ service, method, path string }{
		{"identity", "POST", "/register"},
		{"identity", "POST", "/login"},
		{"identity", "GET", "/me"},
		{"identity", "POST", "/logout"},
		{"identity", "POST", "/change-password"},
		{"identity", "GET", "/admins?page=2&limit=10&search=A%2BB"},
		{"identity", "POST", "/admins"},
		{"identity", "GET", "/admins/test-id"},
		{"identity", "PATCH", "/admins/test-id"},
		{"identity", "GET", "/customers?page=2&status=active"},
		{"identity", "GET", "/customers/test-id"},
		{"identity", "PATCH", "/customers/test-id/status"},
		{"catalog", "GET", "/home/banners"},
		{"catalog", "GET", "/brands"},
		{"catalog", "GET", "/cars?page=2&limit=10&search=A%2BB"},
		{"catalog", "GET", "/cars/test-car"},
		{"catalog", "GET", "/cars/test-car/related"},
		{"catalog", "POST", "/admin/images"},
		{"catalog", "GET", "/admin/brands"},
		{"catalog", "POST", "/admin/brands"},
		{"catalog", "PATCH", "/admin/brands/test-id"},
		{"catalog", "GET", "/admin/banners"},
		{"catalog", "GET", "/admin/banners/1"},
		{"catalog", "PUT", "/admin/banners/1"},
		{"catalog", "DELETE", "/admin/banners/1"},
		{"catalog", "GET", "/admin/cars"},
		{"catalog", "POST", "/admin/cars"},
		{"catalog", "GET", "/admin/cars/test-id"},
		{"catalog", "PATCH", "/admin/cars/test-id"},
	} {
		path := "/api/v1/" + test.service + test.path
		t.Run(test.method+" "+path, func(t *testing.T) {
			response := request(app, httptest.NewRequest(test.method, path, nil))
			want := test.service + " " + test.method + " " + path
			if response.Code != http.StatusOK || response.Body.String() != want {
				t.Fatalf("got status %d and body %q, want 200 and %q", response.Code, response.Body, want)
			}
			if response.Header().Get("X-Test-Ignored") != "" {
				t.Fatal("unexpected upstream response header")
			}
		})
	}
	assertError(t, request(app, httptest.NewRequest("DELETE", "/api/v1/identity/admins/test-id", nil)), 405, "METHOD_NOT_ALLOWED")
	assertError(t, request(app, httptest.NewRequest("GET", "/api/v1/catalog/health", nil)), 404, "NOT_FOUND")
}

func TestForwardsJSONAndAPIResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := testUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil || string(body) != `{"example":true}` || r.Header.Get("Content-Type") != "application/json" ||
			r.Header.Get("Authorization") != "Bearer test-token" || r.Header.Get("X-Request-ID") != "test_1" {
			t.Error("JSON body or allowed headers changed in transit")
		}
		for _, name := range []string{"Cookie", "X-User-ID", "X-Role", "X-Forwarded-For"} {
			if r.Header.Get(name) != "" {
				t.Errorf("unexpected forwarded header %s", name)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Retry-After", "60")
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-Request-ID", r.Header.Get("X-Request-ID"))
		status := map[string]int{"/register": 400, "/login": 429, "/logout": 401, "/change-password": 403, "/admins": 201}[strings.TrimPrefix(r.URL.Path, "/api/v1/identity")]
		w.WriteHeader(status)
		_, _ = io.WriteString(w, `{"request_id":"test_1"}`)
	})
	app := newRouter(upstream, upstream, testLogger())
	for _, test := range []struct {
		path   string
		status int
	}{{"/register", 400}, {"/login", 429}, {"/logout", 401}, {"/change-password", 403}, {"/admins", 201}} {
		req := httptest.NewRequest("POST", "/api/v1/identity"+test.path, strings.NewReader(`{"example":true}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("X-Request-ID", "test_1")
		req.Header.Set("Cookie", "test=value")
		req.Header.Set("X-User-ID", "test-user")
		req.Header.Set("X-Role", "OWNER")
		req.Header.Set("X-Forwarded-For", "192.0.2.1")
		response := request(app, req)
		if response.Code != test.status || response.Body.String() != `{"request_id":"test_1"}` {
			t.Fatalf("%s: status or response body changed", test.path)
		}
		for name, value := range map[string]string{
			"Content-Type": "application/json", "Cache-Control": "no-store", "Retry-After": "60",
			"X-RateLimit-Remaining": "0", "X-Request-ID": "test_1",
		} {
			if response.Header().Get(name) != value {
				t.Errorf("%s: incorrect %s", test.path, name)
			}
		}
	}
}

func TestUploadAndBodyLimits(t *testing.T) {
	gin.SetMode(gin.TestMode)
	file := bytes.Repeat([]byte{0x95}, 5*1024*1024)
	upstream := testUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/catalog/admin/images" {
			t.Error("oversized request reached upstream")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		reader, err := r.MultipartReader()
		if err != nil {
			t.Error("multipart boundary was not forwarded")
			return
		}
		part, err := reader.NextPart()
		if err != nil {
			t.Error("multipart file missing")
			return
		}
		data, err := io.ReadAll(part)
		if err != nil || part.FormName() != "file" || part.FileName() != "test.png" || !bytes.Equal(file, data) {
			t.Error("multipart file changed")
		}
		w.WriteHeader(http.StatusCreated)
	})
	app := newRouter(upstream, upstream, testLogger())
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "test.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(file); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	uploadPath := "/api/v1/catalog/admin/images"
	req := httptest.NewRequest("POST", uploadPath, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if response := request(app, req); response.Code != http.StatusCreated {
		t.Fatalf("upload status = %d, want 201", response.Code)
	}

	for _, test := range []struct {
		path string
		size int
	}{{"/api/v1/identity/login", 16*1024 + 1}, {uploadPath, 6*1024*1024 + 1}} {
		for _, chunked := range []bool{false, true} {
			req := httptest.NewRequest("POST", test.path, strings.NewReader(strings.Repeat("a", test.size)))
			if chunked {
				req.ContentLength = -1
			}
			assertError(t, request(app, req), 413, "PAYLOAD_TOO_LARGE")
		}
	}
}

func TestDependencyFailures(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := testUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/slow":
			w.WriteHeader(http.StatusOK)
			w.(http.Flusher).Flush()
			<-r.Context().Done()
		case "/redirect":
			http.Redirect(w, r, "/other", http.StatusTemporaryRedirect)
		case "/logout":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Error("unexpected request")
		}
	})
	closed := httptest.NewServer(http.NotFoundHandler())
	closedURL, err := url.Parse(closed.URL)
	closed.Close()
	if err != nil {
		t.Fatal(err)
	}
	app := gin.New()
	app.Use(requestHeaders)
	client := newHTTPClient()
	upstreamHandler := forwardTo(client, upstream, "identity", 1024, 200*time.Millisecond, testLogger())
	app.GET("/slow", upstreamHandler)
	app.GET("/redirect", upstreamHandler)
	app.POST("/logout", upstreamHandler)
	app.GET("/offline", forwardTo(client, closedURL, "catalog", 1024, time.Second, testLogger()))

	req := httptest.NewRequest("GET", "/slow", nil)
	req.Header.Set("X-Request-ID", "invalid id")
	response := request(app, req)
	assertError(t, response, 504, "DEPENDENCY_UNAVAILABLE")
	if !validRequestID(response.Header().Get("X-Request-ID")) {
		t.Fatal("invalid request ID was not replaced")
	}
	assertError(t, request(app, httptest.NewRequest("GET", "/offline", nil)), 503, "DEPENDENCY_UNAVAILABLE")
	response = request(app, httptest.NewRequest("GET", "/redirect", nil))
	if response.Code != http.StatusTemporaryRedirect {
		t.Fatalf("redirect status = %d", response.Code)
	}
	response = request(app, httptest.NewRequest("POST", "/logout", nil))
	if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
		t.Fatalf("expected 204 without a response body, got %d", response.Code)
	}
}

func request(app http.Handler, req *http.Request) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	app.ServeHTTP(response, req)
	return response
}

func assertError(t *testing.T, response *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	var body struct {
		RequestID string `json:"request_id"`
		Error     struct{ Code string }
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if response.Code != status || body.Error.Code != code || body.RequestID == "" || body.RequestID != response.Header().Get("X-Request-ID") {
		t.Fatalf("got %d %s with request ID %q, want %d %s with matching request ID", response.Code, body.Error.Code, body.RequestID, status, code)
	}
}

func testUpstream(t *testing.T, handler http.HandlerFunc) *url.URL {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	base, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	return base
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
