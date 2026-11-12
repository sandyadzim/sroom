package storage

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestR2Upload(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/test-bucket/catalog/images/test.png" {
			t.Errorf("unexpected S3 request %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "image/png" || r.ContentLength != int64(len("image bytes")) {
			t.Error("unexpected content type or length")
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 ") {
			t.Error("expected a signed S3 request")
		}
		body, err := io.ReadAll(r.Body)
		if err != nil || string(body) != "image bytes" {
			t.Error("unexpected request body")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	r2, err := NewR2(server.URL, "test-bucket", "test-key", "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	options := r2.client.Options()
	options.HTTPClient = server.Client()
	r2.client = s3.New(options)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := r2.Upload(ctx, "catalog/images/test.png", "image/png", []byte("image bytes")); err != nil {
		t.Fatal(err)
	}
}

func TestR2ImageExists(t *testing.T) {
	for _, test := range []struct {
		status     int
		wantExists bool
		wantError  bool
	}{
		{status: http.StatusOK, wantExists: true},
		{status: http.StatusNotFound},
		{status: http.StatusForbidden, wantError: true},
	} {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodHead || r.URL.Path != "/test-bucket/catalog/images/test.png" {
				t.Errorf("unexpected S3 request %s %s", r.Method, r.URL.Path)
			}
			w.WriteHeader(test.status)
		}))
		r2, err := NewR2(server.URL, "test-bucket", "test-key", "test-secret")
		if err != nil {
			server.Close()
			t.Fatal(err)
		}
		options := r2.client.Options()
		options.HTTPClient = server.Client()
		r2.client = s3.New(options)

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		exists, err := r2.ImageExists(ctx, "catalog/images/test.png")
		cancel()
		server.Close()
		if exists != test.wantExists || (err != nil) != test.wantError {
			t.Errorf("HEAD %d: exists = %t, err = %v", test.status, exists, err)
		}
	}
}
