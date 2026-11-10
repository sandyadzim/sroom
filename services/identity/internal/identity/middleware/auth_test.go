package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"identity/internal/identity/model"
	"identity/internal/identity/service"
	"identity/internal/identity/token"
)

type fakeSessionRepository struct {
	session       *model.Session
	findErr       error
	receivedHash  string
	findCallCount int
}

func (f *fakeSessionRepository) FindByTokenHash(
	ctx context.Context,
	tokenHash string,
) (*model.Session, error) {
	f.findCallCount++
	f.receivedHash = tokenHash

	if f.findErr != nil {
		return nil, f.findErr
	}

	return f.session, nil
}

func setupAuthTestApp(
	repo *fakeSessionRepository,
) *fiber.App {
	app := fiber.New()

	authService := service.NewAuthService(repo)
	authMiddleware := NewAuthMiddleware(authService)

	app.Get(
		"/protected",
		authMiddleware.RequireAuth,
		func(c fiber.Ctx) error {
			session, ok := GetSession(c)
			if !ok {
				return c.SendStatus(fiber.StatusUnauthorized)
			}

			return c.JSON(fiber.Map{
				"user_id": session.UserID.String(),
			})
		},
	)

	return app
}

func TestAuthMiddlewareValidToken(t *testing.T) {
	plainToken := "valid-test-token"
	userID := uuid.New()

	expectedSession := &model.Session{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: token.Hash(plainToken),
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}

	repo := &fakeSessionRepository{
		session: expectedSession,
	}

	app := setupAuthTestApp(repo)

	req := httptest.NewRequest(
		http.MethodGet,
		"/protected",
		nil,
	)
	req.Header.Set(
		fiber.HeaderAuthorization,
		"Bearer "+plainToken,
	)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() returned error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf(
			"status = %d, want %d",
			resp.StatusCode,
			fiber.StatusOK,
		)
	}

	if repo.receivedHash != token.Hash(plainToken) {
		t.Error("repository did not receive the correct token hash")
	}

	if repo.findCallCount != 1 {
		t.Errorf(
			"repository called %d times, want 1",
			repo.findCallCount,
		)
	}
}

func TestAuthMiddlewareMissingAuthorization(t *testing.T) {
	repo := &fakeSessionRepository{}
	app := setupAuthTestApp(repo)

	req := httptest.NewRequest(
		http.MethodGet,
		"/protected",
		nil,
	)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() returned error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf(
			"status = %d, want %d",
			resp.StatusCode,
			fiber.StatusUnauthorized,
		)
	}

	if repo.findCallCount != 0 {
		t.Error("repository must not be called without a token")
	}
}

func TestAuthMiddlewareInvalidHeaderFormat(t *testing.T) {
	tests := []struct {
		name          string
		authorization string
	}{
		{
			name:          "missing bearer prefix",
			authorization: "some-token",
		},
		{
			name:          "wrong scheme",
			authorization: "Basic some-token",
		},
		{
			name:          "bearer without token",
			authorization: "Bearer",
		},
		{
			name:          "too many parts",
			authorization: "Bearer token extra",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeSessionRepository{}
			app := setupAuthTestApp(repo)

			req := httptest.NewRequest(
				http.MethodGet,
				"/protected",
				nil,
			)
			req.Header.Set(
				fiber.HeaderAuthorization,
				tt.authorization,
			)

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test() returned error: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != fiber.StatusUnauthorized {
				t.Errorf(
					"status = %d, want %d",
					resp.StatusCode,
					fiber.StatusUnauthorized,
				)
			}

			if repo.findCallCount != 0 {
				t.Error("repository must not be called for malformed header")
			}
		})
	}
}

func TestAuthMiddlewareInvalidSession(t *testing.T) {
	repo := &fakeSessionRepository{
		findErr: gorm.ErrRecordNotFound,
	}

	app := setupAuthTestApp(repo)

	req := httptest.NewRequest(
		http.MethodGet,
		"/protected",
		nil,
	)
	req.Header.Set(
		fiber.HeaderAuthorization,
		"Bearer unknown-token",
	)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() returned error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf(
			"status = %d, want %d",
			resp.StatusCode,
			fiber.StatusUnauthorized,
		)
	}

	if repo.findCallCount != 1 {
		t.Errorf(
			"repository called %d times, want 1",
			repo.findCallCount,
		)
	}
}

func TestAuthMiddlewareDatabaseError(t *testing.T) {
	repo := &fakeSessionRepository{
		findErr: errors.New("database unavailable"),
	}

	app := setupAuthTestApp(repo)

	req := httptest.NewRequest(
		http.MethodGet,
		"/protected",
		nil,
	)
	req.Header.Set(
		fiber.HeaderAuthorization,
		"Bearer valid-looking-token",
	)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() returned error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf(
			"status = %d, want %d",
			resp.StatusCode,
			fiber.StatusInternalServerError,
		)
	}

	if repo.findCallCount != 1 {
		t.Errorf(
			"repository called %d times, want 1",
			repo.findCallCount,
		)
	}
}

func TestAuthMiddlewareBearerCaseInsensitive(t *testing.T) {
	plainToken := "valid-test-token"

	repo := &fakeSessionRepository{
		session: &model.Session{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			TokenHash: token.Hash(plainToken),
			ExpiresAt: time.Now().UTC().Add(time.Hour),
		},
	}

	app := setupAuthTestApp(repo)

	req := httptest.NewRequest(
		http.MethodGet,
		"/protected",
		nil,
	)
	req.Header.Set(
		fiber.HeaderAuthorization,
		"bearer "+plainToken,
	)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() returned error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf(
			"status = %d, want %d",
			resp.StatusCode,
			fiber.StatusOK,
		)
	}
}
