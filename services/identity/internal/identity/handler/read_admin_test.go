package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"identity/internal/identity/dto"
	"identity/internal/identity/middleware"
	"identity/internal/identity/model"
	"identity/internal/identity/response"
	"identity/internal/identity/service"
	"identity/internal/identity/token"
)

type readAdminUsers struct {
	actor                model.User
	admin                model.User
	err                  error
	calls, limit, offset int
	ctx                  context.Context
	search, status       string
}

func (r *readAdminUsers) FindByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	if id == r.actor.ID {
		return &r.actor, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *readAdminUsers) ListAdmins(ctx context.Context, limit, offset int, search, status string) ([]model.User, int64, error) {
	r.calls++
	r.limit, r.offset, r.ctx = limit, offset, ctx
	r.search, r.status = search, status
	if r.err != nil {
		return nil, 0, r.err
	}
	if r.admin.ID == uuid.Nil {
		return nil, 0, nil
	}
	if offset > 0 {
		return nil, 1, nil
	}
	return []model.User{r.admin}, 1, nil
}
func (r *readAdminUsers) FindAdminByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	r.calls++
	r.ctx = ctx
	if r.err != nil {
		return nil, r.err
	}
	if id != r.admin.ID {
		return nil, gorm.ErrRecordNotFound
	}
	return &r.admin, nil
}

func readAdminApp(users *readAdminUsers) *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: response.ErrorHandler})
	middleware.UseRequests(app, time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	sessions := &testSessions{session: &model.Session{UserID: users.actor.ID, TokenHash: token.Hash("test-token")}}
	auth := middleware.NewAuthMiddleware(service.NewAuthService(sessions))
	h := NewReadAdminHandler(service.NewReadAdminService(users))
	admins := app.Group("/api/v1/identity/admins", auth.RequireAuth, middleware.RequireOwner(users))
	admins.Get("/", h.List)
	admins.Get("/:id", h.Detail)
	return app
}

func TestReadAdminsSuccess(t *testing.T) {
	now := time.Date(2026, 11, 10, 10, 0, 0, 0, time.FixedZone("WITA", 8*60*60))
	users := &readAdminUsers{
		actor: model.User{ID: uuid.New(), Role: model.RoleOwner},
		admin: model.User{ID: uuid.New(), Role: model.RoleAdmin, Status: model.StatusActive, FullName: "Synthetic Admin", Email: "admin@example.invalid", PasswordHash: "private-hash", CreatedAt: now, UpdatedAt: now},
	}
	app := readAdminApp(users)
	body := request(t, app, "GET", "/api/v1/identity/admins", "", "", "test-token", 200)
	var list dto.AdminListResponse
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Data) != 1 || list.Data[0].ID != users.admin.ID.String() || list.Page.Number != 1 || list.Page.Limit != 20 || list.Page.Total != 1 {
		t.Fatal("unexpected list or default pagination")
	}
	if users.limit != 20 || users.offset != 0 {
		t.Fatal("wrong repository pagination")
	}
	if strings.Contains(string(body), "password") || strings.Contains(string(body), "private-hash") {
		t.Fatal("list must not expose password data")
	}
	if _, ok := users.ctx.Deadline(); !ok || users.ctx.Err() != context.Canceled {
		t.Fatal("request context not propagated")
	}
	body = request(t, app, "GET", "/api/v1/identity/admins/"+users.admin.ID.String(), "", "", "test-token", 200)
	var detail struct {
		Data dto.AdminResponse `json:"data"`
	}
	if err := json.Unmarshal(body, &detail); err != nil {
		t.Fatal(err)
	}
	if detail.Data != list.Data[0] || detail.Data.CreatedAt != now.UTC().Format(time.RFC3339) || detail.Data.UpdatedAt != now.UTC().Format(time.RFC3339) {
		t.Fatal("detail must match the safe list DTO and UTC timestamps")
	}
	for _, secret := range []string{"password", "hash", "token", "session", "private-hash"} {
		if strings.Contains(string(body), secret) {
			t.Fatal("unexpected sensitive field")
		}
	}
	for _, query := range []string{"?page=&limit=", "?limit=100"} {
		request(t, app, "GET", "/api/v1/identity/admins"+query, "", "", "test-token", 200)
	}
	body = request(t, app, "GET", "/api/v1/identity/admins?page=3&limit=2", "", "", "test-token", 200)
	if users.limit != 2 || users.offset != 4 {
		t.Fatal("wrong explicit pagination")
	}
	if !strings.Contains(string(body), `"data":[]`) {
		t.Fatal("empty page must return an array")
	}
	if err := json.Unmarshal(body, &list); err != nil || list.Page.Total != 1 {
		t.Fatal("an empty page must retain the total from the repository")
	}
	users.admin = model.User{}
	body = request(t, app, "GET", "/api/v1/identity/admins", "", "", "test-token", 200)
	if err := json.Unmarshal(body, &list); err != nil || len(list.Data) != 0 || list.Page.Total != 0 {
		t.Fatal("empty list must be successful")
	}
}

func TestReadAdminsDeniedAndInvalid(t *testing.T) {
	for _, tt := range []struct {
		name, path, bearer, code string
		role                     model.Role
		err                      error
		status                   int
	}{
		{name: "guest list", path: "", status: 401, code: "UNAUTHENTICATED"},
		{name: "guest detail", path: "/target", status: 401, code: "UNAUTHENTICATED"},
		{name: "customer list", role: model.RoleCustomer, bearer: "test-token", status: 403, code: "FORBIDDEN"},
		{name: "admin detail", path: "/target", role: model.RoleAdmin, bearer: "test-token", status: 403, code: "FORBIDDEN"},
		{name: "admin list", role: model.RoleAdmin, bearer: "test-token", status: 403, code: "FORBIDDEN"},
		{name: "customer detail", path: "/target", role: model.RoleCustomer, bearer: "test-token", status: 403, code: "FORBIDDEN"},
		{name: "invalid token", bearer: "wrong-token", status: 401, code: "UNAUTHENTICATED"},
		{name: "zero page", path: "?page=0", status: 400, code: "INVALID_REQUEST"},
		{name: "negative page", path: "?page=-1", status: 400, code: "INVALID_REQUEST"},
		{name: "invalid page", path: "?page=abc", status: 400, code: "INVALID_REQUEST"},
		{name: "invalid status", path: "?status=disabled", status: 400, code: "INVALID_REQUEST"},
		{name: "invalid limit", path: "?limit=abc", status: 400, code: "INVALID_REQUEST"},
		{name: "zero limit", path: "?limit=0", status: 400, code: "INVALID_REQUEST"},
		{name: "negative limit", path: "?limit=-1", status: 400, code: "INVALID_REQUEST"},
		{name: "limit too large", path: "?limit=101", status: 400, code: "INVALID_REQUEST"},
		{name: "offset overflow", path: "?page=" + strconv.Itoa(math.MaxInt), status: 400, code: "INVALID_REQUEST"},
		{name: "page integer overflow", path: "?page=99999999999999999999999999999", status: 400, code: "INVALID_REQUEST"},
		{name: "invalid ID", path: "/invalid", status: 400, code: "INVALID_REQUEST"},
		{name: "missing ID", path: "/missing", status: 404, code: "NOT_FOUND"},
		{name: "owner target", path: "/owner", status: 404, code: "NOT_FOUND"},
		{name: "customer target", path: "/customer", status: 404, code: "NOT_FOUND"},
		{name: "list error", err: errors.New("private-database-error"), status: 500, code: "INTERNAL_ERROR"},
		{name: "detail error", path: "/target", err: context.DeadlineExceeded, status: 504, code: "REQUEST_TIMEOUT"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			role, bearer := tt.role, tt.bearer
			if tt.status != 401 && tt.status != 403 {
				role, bearer = model.RoleOwner, "test-token"
			}
			users := &readAdminUsers{actor: model.User{ID: uuid.New(), Role: role}, admin: model.User{ID: uuid.New(), Role: model.RoleAdmin}, err: tt.err}
			path := strings.ReplaceAll(tt.path, "/target", "/"+users.admin.ID.String())
			path = strings.ReplaceAll(path, "/missing", "/"+uuid.NewString())
			path = strings.ReplaceAll(path, "/owner", "/"+users.actor.ID.String())
			path = strings.ReplaceAll(path, "/customer", "/"+uuid.NewString())
			body := request(t, readAdminApp(users), "GET", "/api/v1/identity/admins"+path, "", "", bearer, tt.status)
			var result response.ErrorResponse
			if err := json.Unmarshal(body, &result); err != nil || result.Error.Code != tt.code {
				t.Fatal("wrong error envelope")
			}
			if strings.Contains(string(body), "private-database-error") {
				t.Fatal("raw error exposed")
			}
			if tt.status == 401 || tt.status == 403 || tt.status == 400 {
				if users.calls != 0 {
					t.Fatal("denied or invalid requests must not query admin data")
				}
			}
		})
	}
}

func TestReadAdminFilters(t *testing.T) {
	for _, tt := range []struct{ query, search, status string }{
		{query: "?search=%20AnDi%20", search: "AnDi"},
		{query: "?search=admin%40example.invalid", search: "admin@example.invalid"},
		{query: "?status=active", status: model.StatusActive},
		{query: "?search=andi&status=inactive", search: "andi", status: model.StatusInactive},
		{query: "?search=%20&status="},
	} {
		t.Run(tt.query, func(t *testing.T) {
			users := &readAdminUsers{
				actor: model.User{ID: uuid.New(), Role: model.RoleOwner},
				admin: model.User{ID: uuid.New(), Role: model.RoleAdmin, Status: model.StatusInactive},
			}
			body := request(t, readAdminApp(users), "GET", "/api/v1/identity/admins"+tt.query, "", "", "test-token", 200)
			if users.search != tt.search || users.status != tt.status || users.limit != 20 {
				t.Fatal("filters must reach the repository and preserve default pagination")
			}
			var result dto.AdminListResponse
			if err := json.Unmarshal(body, &result); err != nil {
				t.Fatal(err)
			}
			if len(result.Data) != 1 || result.Data[0].Status != model.StatusInactive || result.Page.Total != 1 {
				t.Fatal("response must expose stored status and filtered total from repository")
			}
		})
	}
}
