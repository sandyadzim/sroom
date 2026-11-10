package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"identity/internal/identity/model"
	"identity/internal/identity/response"
	"identity/internal/identity/service"
)

type fakeOwnerUserRepository struct {
	user       *model.User
	err        error
	receivedID uuid.UUID
	ctx        context.Context
	calls      int
}

func (r *fakeOwnerUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	r.receivedID, r.ctx = id, ctx
	r.calls++
	return r.user, r.err
}

func TestRequireOwner(t *testing.T) {
	for _, tt := range []struct {
		name       string
		role       model.Role
		token      string
		sessionErr error
		userErr    error
		status     int
		code       string
	}{
		{name: "owner allowed", role: model.RoleOwner, token: "test-token", status: 204},
		{name: "customer forbidden", role: model.RoleCustomer, token: "test-token", status: 403, code: "FORBIDDEN"},
		{name: "admin forbidden", role: model.RoleAdmin, token: "test-token", status: 403, code: "FORBIDDEN"},
		{name: "unknown role forbidden", role: "UNKNOWN", token: "test-token", status: 403, code: "FORBIDDEN"},
		{name: "guest denied", status: 401, code: "UNAUTHENTICATED"},
		{name: "invalid session denied", token: "test-token", sessionErr: gorm.ErrRecordNotFound, status: 401, code: "UNAUTHENTICATED"},
		{name: "missing user denied", token: "test-token", userErr: gorm.ErrRecordNotFound, status: 401, code: "UNAUTHENTICATED"},
		{name: "database error", token: "test-token", userErr: errors.New("synthetic-private-database-error"), status: 500, code: "INTERNAL_ERROR"},
		{name: "database deadline", token: "test-token", userErr: context.DeadlineExceeded, status: 504, code: "REQUEST_TIMEOUT"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			userID := uuid.New()
			sessions := &fakeSessionRepository{
				session: &model.Session{ID: uuid.New(), UserID: userID},
				findErr: tt.sessionErr,
			}
			users := &fakeOwnerUserRepository{user: &model.User{ID: userID, Role: tt.role}, err: tt.userErr}
			app := fiber.New(fiber.Config{ErrorHandler: response.ErrorHandler})
			UseRequests(app, time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
			auth := NewAuthMiddleware(service.NewAuthService(sessions))
			called := false
			app.Post("/owner-only", auth.RequireAuth, RequireOwner(users), func(c fiber.Ctx) error {
				called = true
				return c.SendStatus(204)
			})
			// Forged client claims must never override the user read from the session.
			req := httptest.NewRequest("POST", "/owner-only", strings.NewReader(`{"role":"OWNER","user_id":"forged"}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Role", "OWNER")
			req.Header.Set("X-User-ID", uuid.NewString())
			req.Header.Set("X-Request-ID", "owner-test")
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			res, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			if res.StatusCode != tt.status || called != (tt.status == 204) {
				t.Fatalf("status=%d handler called=%v, want status=%d", res.StatusCode, called, tt.status)
			}
			if tt.token == "" || tt.sessionErr != nil {
				if users.calls != 0 {
					t.Fatal("user lookup must not run before successful authentication")
				}
			} else {
				if users.calls != 1 || users.receivedID != userID {
					t.Fatal("lookup must use the authenticated session's user ID")
				}
				if _, ok := users.ctx.Deadline(); !ok || users.ctx.Err() != context.Canceled {
					t.Fatal("lookup must receive the request deadline and cancellation")
				}
			}
			if tt.code != "" {
				var body response.ErrorResponse
				if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if body.Error.Code != tt.code || body.RequestID != "owner-test" {
					t.Fatal("unexpected error envelope")
				}
				if strings.Contains(body.Error.Message, "synthetic-private") {
					t.Fatal("raw database error exposed")
				}
			}
		})
	}
}

func TestRequireOwnerWithoutSession(t *testing.T) {
	users := &fakeOwnerUserRepository{}
	app := fiber.New()
	app.Get("/owner-only", RequireOwner(users), func(c fiber.Ctx) error {
		t.Fatal("handler must not run without a session")
		return nil
	})
	res, err := app.Test(httptest.NewRequest("GET", "/owner-only", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 401 || users.calls != 0 {
		t.Fatal("missing session must be denied before user lookup")
	}
}

func TestRequireOwnerReadsRoleOnEveryRequest(t *testing.T) {
	userID := uuid.New()
	users := &fakeOwnerUserRepository{user: &model.User{ID: userID, Role: model.RoleOwner}}
	sessions := &fakeSessionRepository{session: &model.Session{ID: uuid.New(), UserID: userID}}
	auth := NewAuthMiddleware(service.NewAuthService(sessions))
	app := fiber.New()
	app.Get("/owner-only", auth.RequireAuth, RequireOwner(users), func(c fiber.Ctx) error { return c.SendStatus(204) })
	for _, status := range []int{204, 403} {
		req := httptest.NewRequest("GET", "/owner-only", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		res, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != status {
			t.Fatalf("status=%d, want %d", res.StatusCode, status)
		}
		users.user.Role = model.RoleAdmin // Simulate a changed database result.
	}
	if users.calls != 2 {
		t.Fatal("current role must be queried for every request")
	}
}
