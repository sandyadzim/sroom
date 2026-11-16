package router

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"catalog/internal/database"
	"catalog/internal/dto"
	"catalog/internal/handler"
	"catalog/internal/identity"
	"catalog/internal/model"
	"catalog/internal/repository"
	"catalog/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const brandsPath = "/api/v1/catalog/admin/brands"
const missingBrandID = "abfcd0c3-a19a-4714-a5a5-1a91b5dc6ed4"
const uploadedKey = "catalog/images/abcdefghijklmnopqrstuvwxyz.png"
const missingKey = "catalog/images/abcdefghijklmnopqrstuvwxyz.webp"

type testImages struct{}

func (testImages) ImageExists(_ context.Context, key string) (bool, error) {
	if key == "catalog/images/abcdefghijklmnopqrstuvwxyz.jpeg" {
		return false, errors.New("storage unavailable")
	}
	return key == uploadedKey, nil
}

func newBrandTestRouter(t *testing.T, repo *repository.BrandRepository) *gin.Engine {
	t.Helper()
	brandHandler := handler.NewBrandHandler(service.NewBrandService(repo, testImages{}, "https://assets.example.test/"))
	return newCatalogTestRouter(t, brandHandler, handler.NewBannerHandler(nil), handler.NewCarHandler(nil), nil, nil)
}

func newCatalogTestRouter(t *testing.T, brands *handler.BrandHandler, banners *handler.BannerHandler, cars *handler.CarHandler, featuredCars *handler.FeaturedCarHandler, units *handler.CarUnitHandler) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	identityServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if role != "OWNER" && role != "ADMIN" && role != "CUSTOMER" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = io.WriteString(w, `{"data":{"id":"test-user","role":"`+role+`"}}`)
	}))
	t.Cleanup(identityServer.Close)
	client, err := identity.NewClient(identityServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	router, err := New(client, brands, banners, cars, featuredCars, units, nil)
	if err != nil {
		t.Fatal(err)
	}
	return router
}

func brandRequest(t *testing.T, router *gin.Engine, method, path, token, body string, wantStatus int) []byte {
	t.Helper()
	return catalogRequest(t, router, method, brandsPath+path, token, body, wantStatus)
}

func catalogRequest(t *testing.T, router *gin.Engine, method, path, token, body string, wantStatus int) []byte {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != wantStatus {
		t.Fatalf("status = %d, want %d. Body: %s", response.Code, wantStatus, response.Body)
	}
	return response.Body.Bytes()
}

func TestBrandAuthorizationAndValidation(t *testing.T) {
	router := newBrandTestRouter(t, nil)
	for _, test := range []struct {
		name, method, path, token, body string
		status                          int
	}{
		{name: "guest list", method: "GET", status: 401},
		{name: "guest create", method: "POST", body: `{"name":"Toyota"}`, status: 401},
		{name: "invalid session", method: "GET", token: "expired", status: 401},
		{name: "customer list", method: "GET", token: "CUSTOMER", status: 403},
		{name: "customer create", method: "POST", token: "CUSTOMER", body: `{"name":"Toyota"}`, status: 403},
		{name: "customer update", method: "PATCH", path: "/" + missingBrandID, token: "CUSTOMER", body: `{"name":"Toyota"}`, status: 403},
		{name: "bad pagination", method: "GET", path: "?page=0&limit=101", token: "OWNER", status: 400},
		{name: "bad id", method: "PATCH", path: "/bad", token: "ADMIN", body: `{"name":"Toyota"}`, status: 400},
		{name: "oversized brand body", method: "POST", token: "ADMIN", body: `{"name":"Toyota"}` + strings.Repeat(" ", 16*1024), status: 413},
		{name: "blank name", method: "POST", token: "OWNER", body: `{"name":" "}`, status: 400},
		{name: "missing create name", method: "POST", token: "OWNER", body: `{}`, status: 400},
		{name: "non-string create name", method: "POST", token: "OWNER", body: `{"name":123}`, status: 400},
		{name: "long name", method: "POST", token: "OWNER", body: `{"name":"` + strings.Repeat("é", 101) + `"}`, status: 400},
		{name: "null name", method: "PATCH", path: "/" + missingBrandID, token: "OWNER", body: `{"name":null}`, status: 400},
		{name: "unknown status", method: "PATCH", path: "/" + missingBrandID, token: "OWNER", body: `{"status":"archived"}`, status: 400},
		{name: "null status", method: "PATCH", path: "/" + missingBrandID, token: "OWNER", body: `{"status":null}`, status: 400},
		{name: "unknown field", method: "POST", token: "ADMIN", body: `{"name":"Toyota","role":"OWNER"}`, status: 400},
		{name: "trailing JSON", method: "POST", token: "ADMIN", body: `{"name":"Toyota"} {}`, status: 400},
		{name: "empty update", method: "PATCH", path: "/" + missingBrandID, token: "OWNER", body: `{}`, status: 400},
		{name: "URL instead of key", method: "POST", token: "ADMIN", body: `{"name":"Toyota","image_object_key":"https://example.com/image.png"}`, status: 400},
		{name: "missing image", method: "POST", token: "ADMIN", body: `{"name":"Toyota","image_object_key":"` + missingKey + `"}`, status: 400},
		{name: "image storage unavailable", method: "POST", token: "ADMIN", body: `{"name":"Toyota","image_object_key":"catalog/images/abcdefghijklmnopqrstuvwxyz.jpeg"}`, status: 503},
		{name: "filename without a key", method: "POST", token: "ADMIN", body: `{"name":"Toyota","image_filename":"logo.png"}`, status: 400},
		{name: "filename with a path", method: "POST", token: "ADMIN", body: `{"name":"Toyota","image_object_key":"` + uploadedKey + `","image_filename":"folder/logo.png"}`, status: 400},
		{name: "filename on detach", method: "PATCH", path: "/" + missingBrandID, token: "ADMIN", body: `{"image_object_key":null,"image_filename":"logo.png"}`, status: 400},
	} {
		t.Run(test.name, func(t *testing.T) {
			brandRequest(t, router, test.method, test.path, test.token, test.body, test.status)
		})
	}
}

func TestBrandPostgres(t *testing.T) {
	db := catalogTestDatabase(t)
	repo := repository.NewBrandRepository(db)
	router := newBrandTestRouter(t, repo)

	empty := brandRequest(t, router, "GET", "", "ADMIN", "", 200)
	var page dto.BrandListResponse
	if err := json.Unmarshal(empty, &page); err != nil {
		t.Fatal(err)
	}
	if page.Data == nil || len(page.Data) != 0 || page.Page != (dto.BrandPage{Number: 1, Limit: 20, Total: 0}) {
		t.Fatalf("unexpected empty list: %s", empty)
	}
	created := brandRequest(t, router, "POST", "", "OWNER", `{"name":" Toyota ","image_object_key":"`+uploadedKey+`","image_filename":"Toyota Logo.png"}`, 201)
	var result struct {
		Message string            `json:"message"`
		Data    dto.BrandResponse `json:"data"`
	}
	if err := json.Unmarshal(created, &result); err != nil {
		t.Fatal(err)
	}
	item := result.Data
	if result.Message != "Brand created successfully" || item.Status != "active" ||
		uuid.Validate(item.ID) != nil || item.Name != "Toyota" || item.ImageObjectKey == nil || *item.ImageObjectKey != uploadedKey {
		t.Fatalf("unexpected created brand: %s", created)
	}
	if !strings.HasSuffix(item.CreatedAt, "Z") || !strings.HasSuffix(item.UpdatedAt, "Z") {
		t.Fatalf("expected UTC timestamps: %s", created)
	}
	if item.ImageFilename == nil || *item.ImageFilename != "Toyota Logo.png" ||
		item.ImageURL == nil || *item.ImageURL != "https://assets.example.test/"+uploadedKey {
		t.Fatalf("expected filename and public URL: %s", created)
	}
	path := "/" + item.ID
	brandRequest(t, router, "POST", "", "ADMIN", `{"name":" toyOTA "}`, 409)
	brandRequest(t, router, "POST", "", "ADMIN", `{"name":"Acme_100%"}`, 201)
	brandRequest(t, router, "PATCH", path, "ADMIN", `{"name":"acme_100%"}`, 409)
	brandRequest(t, router, "PATCH", "/"+missingBrandID, "OWNER", `{"name":"Missing"}`, 404)

	inactive := brandRequest(t, router, "PATCH", path, "ADMIN", `{"status":"inactive"}`, 200)
	if err := json.Unmarshal(inactive, &result); err != nil {
		t.Fatal(err)
	}
	if result.Message != "Brand updated successfully" || result.Data.Status != "inactive" ||
		result.Data.Name != "Toyota" || result.Data.ImageObjectKey == nil || *result.Data.ImageObjectKey != uploadedKey {
		t.Fatalf("expected deactivation to preserve name and image: %s", inactive)
	}

	found := brandRequest(t, router, "GET", "?search=TOYO&limit=1", "OWNER", "", 200)
	if !strings.Contains(string(found), `"name":"Toyota"`) || !strings.Contains(string(found), `"total":1`) {
		t.Fatalf("unexpected search response: %s", found)
	}
	literal := brandRequest(t, router, "GET", "?search=%25&limit=1", "ADMIN", "", 200)
	if !strings.Contains(string(literal), `"name":"Acme_100%"`) || !strings.Contains(string(literal), `"total":1`) {
		t.Fatalf("expected literal percent search: %s", literal)
	}
	secondPage := brandRequest(t, router, "GET", "?page=2&limit=1", "ADMIN", "", 200)
	if err := json.Unmarshal(secondPage, &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Data) != 1 || page.Data[0].Name != "Toyota" || page.Data[0].Status != "inactive" ||
		page.Page != (dto.BrandPage{Number: 2, Limit: 1, Total: 2}) {
		t.Fatalf("unexpected page 2: %s", secondPage)
	}
	reactivated := brandRequest(t, router, "PATCH", path, "OWNER", `{"status":"active"}`, 200)
	if !strings.Contains(string(reactivated), `"status":"active"`) {
		t.Fatalf("expected active status: %s", reactivated)
	}
	brandRequest(t, router, "PATCH", path, "ADMIN", `{"image_object_key":"`+missingKey+`"}`, 400)
	updated := brandRequest(t, router, "PATCH", path, "ADMIN", `{"name":"Toyota Motor"}`, 200)
	if !strings.Contains(string(updated), `"name":"Toyota Motor"`) || !strings.Contains(string(updated), `"image_filename":"Toyota Logo.png"`) {
		t.Fatalf("expected rename to preserve image: %s", updated)
	}
	cleared := brandRequest(t, router, "PATCH", path, "OWNER", `{"image_object_key":null}`, 200)
	if !strings.Contains(string(cleared), `"name":"Toyota Motor"`) ||
		!strings.Contains(string(cleared), `"image_object_key":null`) ||
		!strings.Contains(string(cleared), `"image_filename":null`) ||
		!strings.Contains(string(cleared), `"image_url":null`) {
		t.Fatalf("expected image cleared and name preserved: %s", cleared)
	}

	results := make(chan error, 2)
	for _, name := range []string{"Concurrent Brand", "concurrent brand"} {
		go func() {
			_, err := repo.Create(context.Background(), model.Brand{Name: name})
			results <- err
		}()
	}
	first, second := <-results, <-results
	if !((first == nil && errors.Is(second, repository.ErrBrandNameTaken)) ||
		(second == nil && errors.Is(first, repository.ErrBrandNameTaken))) {
		t.Fatalf("expected one concurrent insert and one duplicate error, got %v and %v", first, second)
	}
	if _, err := repo.Create(context.Background(), model.Brand{Name: strings.Repeat("é", 101)}); err == nil {
		t.Fatal("expected PostgreSQL to reject a brand name over 100 characters")
	}
	if err := db.Exec("UPDATE brands SET status = ? WHERE id = ?", "archived", item.ID).Error; err == nil {
		t.Fatal("expected PostgreSQL to reject an invalid status")
	}
}

func catalogTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("CATALOG_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set CATALOG_TEST_DATABASE_URL to a PostgreSQL test database")
	}
	base, err := database.Connect(dsn)
	if err != nil {
		t.Fatal(err)
	}
	basePool, err := base.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = basePool.Close() })

	schema := "catalog_test_" + strings.ToLower(rand.Text())
	if err := base.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := base.Exec("DROP SCHEMA " + schema + " CASCADE").Error; err != nil {
			t.Error(err)
		}
	})
	testURL, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := testURL.Query()
	query.Set("search_path", schema)
	testURL.RawQuery = query.Encode()
	db, err := database.Connect(testURL.String())
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pool.Close() })
	for _, file := range []string{
		"000001_create_brands.up.sql",
		"000002_limit_brand_name.up.sql",
		"000003_add_brand_status.up.sql",
		"000004_create_banners.up.sql",
		"000005_create_cars.up.sql",
		"000006_create_car_images.up.sql",
		"000007_add_brand_image_filename.up.sql",
		"000008_order_car_images.up.sql",
		"000009_link_banners_to_cars.up.sql",
		"000010_create_featured_cars.up.sql",
		"000011_create_car_units.up.sql",
	} {
		sql, err := os.ReadFile("../../migrations/" + file)
		if err != nil {
			t.Fatal(err)
		}
		if err := db.Exec(string(sql)).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}
