package router

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"catalog/internal/handler"
	"catalog/internal/identity"

	"github.com/gin-gonic/gin"
)

const imageUploadPath = "/api/v1/catalog/admin/images"
const maxImageBytes = 5 * 1024 * 1024
const maxUploadBodyBytes = 6 * 1024 * 1024

type uploadFunc func(context.Context, string, string, []byte) error

func (upload uploadFunc) Upload(ctx context.Context, key, contentType string, data []byte) error {
	return upload(ctx, key, contentType, data)
}

func TestImageUpload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var pngFile bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.NoCompression}
	if err := encoder.Encode(&pngFile, image.NewRGBA(image.Rect(0, 0, 128, 128))); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name           string
		role           string
		identityStatus int
		noToken        bool
		badProfile     bool
		identitySlow   bool
		badFile        bool
		fileSize       int
		extraPart      bool
		unknownLength  bool
		storageFails   bool
		wantStatus     int
		wantUpload     bool
	}{
		{name: "owner uploads PNG", role: "OWNER", wantStatus: 201, wantUpload: true},
		{name: "admin uploads PNG", role: "ADMIN", wantStatus: 201, wantUpload: true},
		{name: "guest", noToken: true, wantStatus: 401},
		{name: "customer with forged role header", role: "CUSTOMER", wantStatus: 403},
		{name: "invalid or expired token", identityStatus: 401, wantStatus: 401},
		{name: "identity unavailable", identityStatus: 500, wantStatus: 503},
		{name: "identity redirect", identityStatus: 302, wantStatus: 503},
		{name: "identity request deadline", identitySlow: true, wantStatus: 503},
		{name: "malformed profile", badProfile: true, wantStatus: 503},
		{name: "unknown role", role: "OTHER", wantStatus: 503},
		{name: "fake image", role: "ADMIN", badFile: true, wantStatus: 415},
		{name: "file over 5 MiB", role: "ADMIN", fileSize: maxImageBytes + 1, wantStatus: 413},
		{name: "request over 6 MiB", role: "ADMIN", fileSize: maxUploadBodyBytes, wantStatus: 413},
		{name: "extra multipart file", role: "ADMIN", extraPart: true, wantStatus: 400},
		{name: "chunked body over 6 MiB", role: "ADMIN", unknownLength: true, wantStatus: 413},
		{name: "storage failure", role: "ADMIN", storageFails: true, wantStatus: 503, wantUpload: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			identityServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if test.noToken || r.Method != http.MethodGet || r.URL.Path != "/api/v1/identity/me" {
					t.Errorf("unexpected Identity request %s %s", r.Method, r.URL.Path)
				}
				if r.Header.Get("Authorization") != "Bearer test-session" || r.Header.Get("X-Request-ID") == "" {
					t.Error("Identity should receive the token and request ID")
				}
				if test.identityStatus != 0 {
					if test.identityStatus == http.StatusFound {
						w.Header().Set("Location", "/unexpected-redirect")
					}
					w.WriteHeader(test.identityStatus)
					return
				}
				if test.badProfile {
					_, _ = io.WriteString(w, `{"data":`)
					return
				}
				if test.identitySlow {
					<-r.Context().Done()
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]string{"id": "test-user", "role": test.role}})
			}))
			defer identityServer.Close()
			client, err := identity.NewClient(identityServer.URL)
			if err != nil {
				t.Fatal(err)
			}

			var storedKey string
			store := uploadFunc(func(ctx context.Context, key, contentType string, data []byte) error {
				storedKey = key
				if !strings.HasPrefix(key, "catalog/images/") || !strings.HasSuffix(key, ".png") || contentType != "image/png" || !bytes.Equal(data, pngFile.Bytes()) {
					t.Error("storage received an unexpected object")
				}
				if _, ok := ctx.Deadline(); !ok {
					t.Error("upload must have a deadline")
				}
				if test.storageFails {
					return errors.New("synthetic storage detail")
				}
				return nil
			})

			router, err := New(client, handler.NewBrandHandler(nil), handler.NewBannerHandler(nil), handler.NewCarHandler(nil), nil, nil, store)
			if err != nil {
				t.Fatal(err)
			}
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			part, err := writer.CreateFormFile("file", "picture.txt")
			if err != nil {
				t.Fatal(err)
			}
			file := pngFile.Bytes()
			if test.badFile {
				file = []byte("not an image")
			} else if test.fileSize > 0 {
				file = bytes.Repeat([]byte{'a'}, test.fileSize)
			}
			if _, err := part.Write(file); err != nil {
				t.Fatal(err)
			}
			if test.extraPart {
				if _, err := writer.CreateFormFile("file", "another.png"); err != nil {
					t.Fatal(err)
				}
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			if test.unknownLength {
				body.Write(bytes.Repeat([]byte{'x'}, maxUploadBodyBytes))
			}

			request := httptest.NewRequest(http.MethodPost, imageUploadPath, &body)
			request.Header.Set("Content-Type", writer.FormDataContentType())
			request.Header.Set("X-User-Role", "OWNER")
			if !test.noToken {
				request.Header.Set("Authorization", "Bearer test-session")
			}
			if test.unknownLength {
				request.ContentLength = -1
			}
			if test.identitySlow {
				ctx, cancel := context.WithTimeout(request.Context(), 50*time.Millisecond)
				defer cancel()
				request = request.WithContext(ctx)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.wantStatus || (storedKey != "") != test.wantUpload {
				t.Fatalf("status=%d uploaded=%t body=%s", response.Code, storedKey != "", response.Body)
			}
			if test.wantStatus == http.StatusCreated && !strings.Contains(response.Body.String(), storedKey) {
				t.Error("response must return the stored object key")
			}
			if strings.Contains(response.Body.String(), "synthetic storage detail") || strings.Contains(response.Body.String(), "test-session") {
				t.Error("response contains internal details")
			}
		})
	}
}
