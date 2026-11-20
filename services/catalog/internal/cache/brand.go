package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"catalog/internal/dto"

	"github.com/redis/go-redis/v9"
)

const brandTTL = 60 * time.Second
const redisTimeout = 500 * time.Millisecond
const brandsKey = "sroom:catalog:v1:brands:all"
const activeBrandsKey = "sroom:catalog:v1:brands:active"

type BrandCache struct {
	client *redis.Client
}

func NewBrandCache(rawURL string) (*BrandCache, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, nil
	}
	invalidURL := errors.New("REDIS_URL must be a valid redis:// or rediss:// connection URL")
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "redis" && parsed.Scheme != "rediss") || parsed.Hostname() == "" || parsed.Fragment != "" {
		return nil, invalidURL
	}
	options, err := redis.ParseURL(rawURL)
	if err != nil {
		return nil, invalidURL
	}
	options.MaxRetries = -1
	options.DialerRetries = 1
	options.DialTimeout = redisTimeout
	options.ReadTimeout = redisTimeout
	options.WriteTimeout = redisTimeout
	options.PoolTimeout = redisTimeout
	options.ContextTimeoutEnabled = true
	return &BrandCache{client: redis.NewClient(options)}, nil
}

func (c *BrandCache) Get(ctx context.Context, activeOnly bool) ([]dto.CarBrandOption, bool) {
	ctx, cancel := context.WithTimeout(ctx, redisTimeout)
	defer cancel()
	key := brandsKey
	if activeOnly {
		key = activeBrandsKey
	}
	body, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false
	}
	if err != nil {
		slog.Warn("brand cache read unavailable", "service", "catalog", "operation", "brands.cache.get")
		return nil, false
	}
	var brands []dto.CarBrandOption
	if json.Unmarshal(body, &brands) != nil || brands == nil {
		return nil, false
	}
	return brands, true
}

func (c *BrandCache) Set(ctx context.Context, activeOnly bool, brands []dto.CarBrandOption) {
	body, err := json.Marshal(brands)
	if err != nil {
		slog.Warn("brand cache encoding failed", "service", "catalog", "operation", "brands.cache.set")
		return
	}
	key := brandsKey
	if activeOnly {
		key = activeBrandsKey
	}
	ctx, cancel := context.WithTimeout(ctx, redisTimeout)
	defer cancel()
	if err := c.client.Set(ctx, key, body, brandTTL).Err(); err != nil {
		slog.Warn("brand cache write unavailable", "service", "catalog", "operation", "brands.cache.set")
	}
}

func (c *BrandCache) Invalidate(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, redisTimeout)
	defer cancel()
	if err := c.client.Del(ctx, brandsKey, activeBrandsKey).Err(); err != nil {
		slog.Warn("brand cache invalidation unavailable", "service", "catalog", "operation", "brands.cache.invalidate")
	}
}

func (c *BrandCache) Close() error {
	return c.client.Close()
}
