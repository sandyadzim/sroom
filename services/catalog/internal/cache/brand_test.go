package cache

import (
	"strings"
	"testing"
)

func TestBrandCacheConfiguration(t *testing.T) {
	for _, rawURL := range []string{"", "redis://localhost:6379/0", "redis://test-user:test-password@localhost:6379/1", "rediss://localhost:6379/1"} {
		client, err := NewBrandCache(rawURL)
		if err != nil || (rawURL == "") != (client == nil) {
			t.Fatalf("unexpected cache configuration result: %v", err)
		}
		if client != nil {
			if err := client.Close(); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, rawURL := range []string{"http://user:test-password@localhost", "redis://", "redis://localhost/not-a-db", "redis://localhost:bad", "redis://localhost/0#fragment"} {
		client, err := NewBrandCache(rawURL)
		if err == nil || client != nil || strings.Contains(err.Error(), "test-password") {
			t.Fatal("invalid Redis URL was accepted or exposed credentials")
		}
	}
}
