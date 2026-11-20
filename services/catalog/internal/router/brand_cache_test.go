package router

import (
	"context"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"catalog/internal/cache"
	"catalog/internal/dto"
	"catalog/internal/handler"
	"catalog/internal/model"
	"catalog/internal/repository"
	"catalog/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const publicBrandsPath = "/api/v1/catalog/brands"
const brandsCacheKey = "sroom:catalog:v1:brands:all"
const activeBrandsCacheKey = "sroom:catalog:v1:brands:active"

func catalogTestBrandCache(t *testing.T) (*cache.BrandCache, *redis.Client) {
	t.Helper()
	rawURL := os.Getenv("CATALOG_TEST_REDIS_URL")
	if rawURL == "" {
		t.Skip("set CATALOG_TEST_REDIS_URL to a dedicated Redis test database")
	}
	options, err := redis.ParseURL(rawURL)
	if err != nil {
		t.Fatal("invalid Redis test URL")
	}
	observer := redis.NewClient(options)
	client, err := cache.NewBrandCache(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := observer.Del(ctx, brandsCacheKey, activeBrandsCacheKey).Err(); err != nil {
			t.Error("could not clean test cache")
		}
		if err := client.Close(); err != nil {
			t.Error(err)
		}
		if err := observer.Close(); err != nil {
			t.Error(err)
		}
	})
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	if err := observer.Del(ctx, brandsCacheKey, activeBrandsCacheKey).Err(); err != nil {
		t.Fatal("could not connect to the dedicated Redis test database")
	}
	return client, observer
}

func newBrandCacheRouter(t *testing.T, db *gorm.DB, brandCache *cache.BrandCache) *gin.Engine {
	t.Helper()
	brands := handler.NewBrandHandler(service.NewBrandService(repository.NewBrandRepository(db), testImages{}, "https://assets.example.test", brandCache))
	cars := handler.NewCarHandler(service.NewCarService(repository.NewCarRepository(db), carTestImages{}, "https://assets.example.test", brandCache))
	return newCatalogTestRouter(t, brands, handler.NewBannerHandler(nil), cars, nil, nil)
}

func TestPublicBrandCachePostgresRedis(t *testing.T) {
	db := catalogTestDatabase(t)
	brandCache, observer := catalogTestBrandCache(t)
	router := newBrandCacheRouter(t, db, brandCache)
	ctx := t.Context()
	brandQueries := 0
	if err := db.Callback().Query().After("gorm:query").Register("test:count_brand_reads", func(query *gorm.DB) {
		if query.Statement.Table == "brands" {
			brandQueries++
		}
	}); err != nil {
		t.Fatal(err)
	}
	read := func(activeOnly bool) []dto.CarBrandOption {
		t.Helper()
		path := publicBrandsPath
		if activeOnly {
			path += "?status=active"
		}
		var response struct{ Data []dto.CarBrandOption }
		decodeCarJSON(t, catalogRequest(t, router, "GET", path, "", "", 200), &response)
		if response.Data == nil {
			t.Fatal("public brands must return an array, including for empty results")
		}
		return response.Data
	}
	warmBoth := func() {
		t.Helper()
		read(false)
		read(true)
		if count, err := observer.Exists(ctx, brandsCacheKey, activeBrandsCacheKey).Result(); err != nil || count != 2 {
			t.Fatalf("cache keys=%d error=%v", count, err)
		}
	}
	assertInvalidated := func() {
		t.Helper()
		if count, err := observer.Exists(ctx, brandsCacheKey, activeBrandsCacheKey).Result(); err != nil || count != 0 {
			t.Fatal("a successful mutation did not invalidate both keys")
		}
	}

	warmBoth()
	queries := brandQueries
	if len(read(false)) != 0 || brandQueries != queries {
		t.Fatal("an empty cache hit queried PostgreSQL")
	}
	for _, key := range []string{brandsCacheKey, activeBrandsCacheKey} {
		if ttl, err := observer.PTTL(ctx, key).Result(); err != nil || ttl <= 0 || ttl > 60*time.Second {
			t.Fatalf("unexpected TTL: %s (%v)", ttl, err)
		}
	}
	var brand struct{ Data dto.BrandResponse }
	decodeCarJSON(t, catalogRequest(t, router, "POST", brandsPath, "ADMIN", `{"name":"Toyota"}`, 201), &brand)
	assertInvalidated()
	warmBoth()
	active := model.CarActive
	request := carTestRequest(brand.Data.ID, "Innova", 600000)
	request.Status = &active
	car := createTestCar(t, router, request)
	assertInvalidated()
	warmBoth()
	queries = brandQueries
	if data := read(false); len(data) != 1 || data[0].ID != brand.Data.ID || brandQueries != queries {
		t.Fatal("a brand cache hit changed data or queried PostgreSQL")
	}

	// Failed writes must leave the existing cache intact.
	catalogRequest(t, router, "POST", brandsPath, "ADMIN", `{"name":"Toyota"}`, 409)
	catalogRequest(t, router, "POST", adminCarsPath, "ADMIN", carJSON(t, request), 409)
	if count, err := observer.Exists(ctx, brandsCacheKey, activeBrandsCacheKey).Result(); err != nil || count != 2 {
		t.Fatal("a failed write invalidated the cache")
	}
	brandPath := brandsPath + "/" + brand.Data.ID
	for _, body := range []string{`{"name":"Toyota updated"}`, `{"image_object_key":"` + uploadedKey + `"}`} {
		catalogRequest(t, router, "PATCH", brandPath, "ADMIN", body, 200)
		assertInvalidated()
		warmBoth()
	}
	if data := read(false); data[0].Name != "Toyota updated" || data[0].ImageURL == nil || !strings.HasSuffix(*data[0].ImageURL, uploadedKey) {
		t.Fatal("updated name or logo remained stale")
	}
	catalogRequest(t, router, "PATCH", brandPath, "ADMIN", `{"status":"inactive"}`, 200)
	assertInvalidated()
	if len(read(false)) != 1 || len(read(true)) != 0 {
		t.Fatal("active-only and unfiltered brand keys were mixed")
	}
	catalogRequest(t, router, "PATCH", brandPath, "ADMIN", `{"status":"active"}`, 200)
	warmBoth()
	carPath := adminCarsPath + "/" + car.ID
	catalogRequest(t, router, "PATCH", carPath, "ADMIN", `{"status":"inactive"}`, 200)
	assertInvalidated()
	if len(read(false)) != 0 {
		t.Fatal("brand with no active cars remained cached")
	}
	catalogRequest(t, router, "PATCH", carPath, "ADMIN", `{"status":"active"}`, 200)
	if len(read(false)) != 1 {
		t.Fatal("reactivating a car did not refresh the public brand list")
	}

	// Corrupt data is a miss, and a real Redis expiry also refills from PostgreSQL.
	if err := observer.Set(ctx, brandsCacheKey, "invalid JSON", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	queries = brandQueries
	if len(read(false)) != 1 || brandQueries != queries+1 {
		t.Fatal("corrupt cache did not fall back to PostgreSQL")
	}
	if err := observer.PExpire(ctx, brandsCacheKey, time.Millisecond).Err(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(5 * time.Millisecond)
	queries = brandQueries
	if len(read(false)) != 1 || brandQueries != queries+1 {
		t.Fatal("expired cache did not refill from PostgreSQL")
	}
}

func TestUnavailableBrandCacheFallsBackPostgres(t *testing.T) {
	db := catalogTestDatabase(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	brandCache, err := cache.NewBrandCache("redis://" + address + "/0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := brandCache.Close(); err != nil {
			t.Error(err)
		}
	}()
	router := newBrandCacheRouter(t, db, brandCache)
	start := time.Now()
	catalogRequest(t, router, "POST", brandsPath, "ADMIN", `{"name":"Fallback brand"}`, 201)
	var response struct{ Data []dto.CarBrandOption }
	decodeCarJSON(t, catalogRequest(t, router, "GET", publicBrandsPath, "", "", 200), &response)
	if response.Data == nil || len(response.Data) != 0 || time.Since(start) > 3*time.Second {
		t.Fatal("unavailable cache broke or delayed normal PostgreSQL access")
	}
	var count int64
	if err := db.Model(&model.Brand{}).Where("name = ?", "Fallback brand").Count(&count).Error; err != nil || count != 1 {
		t.Fatal("cache failure rolled back the database mutation")
	}
}

func TestDatabaseErrorIsNotCachedAsEmpty(t *testing.T) {
	db := catalogTestDatabase(t)
	brandCache, observer := catalogTestBrandCache(t)
	router := newBrandCacheRouter(t, db, brandCache)
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.Close(); err != nil {
		t.Fatal(err)
	}
	catalogRequest(t, router, "GET", publicBrandsPath, "", "", 500)
	if count, err := observer.Exists(t.Context(), brandsCacheKey).Result(); err != nil || count != 0 {
		t.Fatal("database error was cached as an empty successful response")
	}
}
