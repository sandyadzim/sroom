package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"identity/internal/identity/dto"
	"identity/internal/identity/middleware"
	"identity/internal/identity/model"
	"identity/internal/identity/password"
	"identity/internal/identity/response"
	"identity/internal/identity/service"
	"identity/internal/identity/token"
)

type accountUsers struct {
	users    map[uuid.UUID]model.User
	writes   int
	writeErr error
	ctx      context.Context
}

func (r *accountUsers) FindByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return &user, nil
}

func (r *accountUsers) FindByEmail(_ context.Context, email string) (*model.User, error) {
	for _, user := range r.users {
		if strings.EqualFold(user.Email, email) {
			return &user, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *accountUsers) Create(ctx context.Context, user *model.User) error {
	r.writes++
	r.ctx = ctx
	if r.writeErr != nil {
		return r.writeErr
	}
	if _, err := r.FindByEmail(ctx, user.Email); err == nil {
		return gorm.ErrDuplicatedKey
	}
	user.CreatedAt, user.UpdatedAt = time.Now().UTC(), time.Now().UTC()
	r.users[user.ID] = *user
	return nil
}

func (r *accountUsers) UpdateAdmin(ctx context.Context, id uuid.UUID, changes map[string]any) (*model.User, error) {
	r.writes++
	r.ctx = ctx
	if r.writeErr != nil {
		return nil, r.writeErr
	}
	user, ok := r.users[id]
	if !ok || user.Role != model.RoleAdmin {
		return nil, gorm.ErrRecordNotFound
	}
	if email, ok := changes["email"]; ok {
		if other, err := r.FindByEmail(ctx, email.(string)); err == nil && other.ID != id {
			return nil, gorm.ErrDuplicatedKey
		}
	}
	for field, value := range changes {
		switch field {
		case "full_name":
			user.FullName = value.(string)
		case "email":
			user.Email = value.(string)
		case "status":
			user.Status = value.(string)
		case "password_hash":
			user.PasswordHash = value.(string)
		}
	}
	user.UpdatedAt = time.Now().UTC()
	r.users[id] = user
	return &user, nil
}

func (r *accountUsers) UpdatePassword(ctx context.Context, id uuid.UUID, role model.Role, hash string) error {
	r.writes++
	r.ctx = ctx
	if r.writeErr != nil {
		return r.writeErr
	}
	user, ok := r.users[id]
	if !ok || user.Role != role {
		return gorm.ErrRecordNotFound
	}
	user.PasswordHash = hash
	user.UpdatedAt = time.Now().UTC()
	r.users[id] = user
	return nil
}

type accountSessions struct{ sessions map[string]*model.Session }

func (r *accountSessions) Create(_ context.Context, session *model.Session) error {
	r.sessions[session.TokenHash] = session
	return nil
}
func (r *accountSessions) FindByTokenHash(_ context.Context, hash string) (*model.Session, error) {
	session := r.sessions[hash]
	if session == nil || !session.ExpiresAt.After(time.Now()) {
		return nil, gorm.ErrRecordNotFound
	}
	return session, nil
}
func (r *accountSessions) DeleteByID(_ context.Context, id uuid.UUID) error {
	for hash, session := range r.sessions {
		if session.ID == id {
			delete(r.sessions, hash)
		}
	}
	return nil
}

func accountTestApp() (*fiber.App, *accountUsers, *accountSessions) {
	users := &accountUsers{users: make(map[uuid.UUID]model.User)}
	sessions := &accountSessions{sessions: make(map[string]*model.Session)}
	for bearer, role := range map[string]model.Role{"owner-token": model.RoleOwner, "admin-token": model.RoleAdmin, "customer-token": model.RoleCustomer} {
		id := uuid.New()
		users.users[id] = model.User{ID: id, Role: role, Status: model.StatusActive, Email: strings.ToLower(string(role)) + "@example.invalid"}
		sessions.sessions[token.Hash(bearer)] = &model.Session{ID: uuid.New(), UserID: id, TokenHash: token.Hash(bearer), ExpiresAt: time.Now().Add(time.Hour)}
	}
	app := fiber.New(fiber.Config{ErrorHandler: response.ErrorHandler, BodyLimit: middleware.MaxBodyBytes})
	middleware.UseRequests(app, middleware.DefaultRequestTimeout, slog.New(slog.NewTextHandler(io.Discard, nil)))
	auth := middleware.NewAuthMiddleware(service.NewAuthService(sessions))
	admin := NewAdminHandler(service.NewCreateAdminService(users), service.NewUpdateAdminService(users))
	group := app.Group("/api/v1/identity")
	admins := group.Group("/admins", auth.RequireAuth, middleware.RequireOwner(users))
	admins.Post("/", admin.Create)
	admins.Patch("/:id", admin.Update)
	customers := group.Group("/customers", auth.RequireAuth, middleware.RequireStaff(users))
	customer := NewCustomerHandler(service.NewCustomerService(users))
	customers.Get("/", customer.List)
	customers.Post("/lookup", customer.Lookup)
	customers.Get("/:id", customer.Detail)
	customers.Patch("/:id/status", customer.UpdateStatus)
	group.Get("/me", auth.RequireAuth, NewMeHandler(service.NewMeService(users)).Me)
	group.Post("/logout", auth.RequireAuth, NewLogoutHandler(service.NewLogoutService(sessions)).Logout)
	changePassword := NewChangePasswordHandler(service.NewChangePasswordService(users))
	group.Post("/change-password", auth.RequireAuth, changePassword.Change)
	group.Post("/customer/change-password", auth.RequireAuth, changePassword.ChangeCustomer)
	app.Post("/api/v1/identity/login", NewLoginHandler(service.NewLoginService(users, sessions)).Login)
	return app, users, sessions
}

func accountLogin(t *testing.T, app *fiber.App, email, pass string, status int) string {
	t.Helper()
	data, err := json.Marshal(dto.LoginRequest{Email: email, Password: pass})
	if err != nil {
		t.Fatal(err)
	}
	body := request(t, app, "POST", "/api/v1/identity/login", string(data), "application/json", "", status)
	if status != 200 {
		assertError(t, body, "INVALID_CREDENTIALS")
		return ""
	}
	var result struct {
		Data dto.LoginResponse `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatal(err)
	}
	if result.Data.Token == "" {
		t.Fatal("login must issue a token")
	}
	return result.Data.Token
}

func adminResult(t *testing.T, body []byte) dto.AdminResponse {
	t.Helper()
	var result struct {
		Data dto.AdminResponse `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"password", "hash", "token", "session"} {
		if strings.Contains(string(body), field) {
			t.Fatal("admin response contains sensitive fields")
		}
	}
	return result.Data
}

func TestAdminCreateUpdateLifecycle(t *testing.T) {
	app, users, sessions := accountTestApp()
	beforeSessions := len(sessions.sessions)
	body := request(t, app, "POST", "/api/v1/identity/admins", `{"full_name":"  Synthetic Admin  ","email":" NEW@EXAMPLE.INVALID ","password":"synthetic-old-password"}`, "application/json", "owner-token", 201)
	created := adminResult(t, body)
	id, err := uuid.Parse(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if created.Role != model.RoleAdmin || created.Status != model.StatusActive || created.FullName != "Synthetic Admin" || created.Email != "new@example.invalid" {
		t.Fatal("create must normalize fields and set ADMIN/active on server")
	}
	if len(sessions.sessions) != beforeSessions {
		t.Fatal("create must not log in the new account")
	}
	if valid, err := password.Verify("synthetic-old-password", users.users[id].PasswordHash); err != nil || !valid {
		t.Fatal("password not hashed correctly")
	}
	if _, ok := users.ctx.Deadline(); !ok || users.ctx.Err() != context.Canceled {
		t.Fatal("request context not propagated to mutation")
	}
	oldToken := accountLogin(t, app, created.Email, "synthetic-old-password", 200)
	request(t, app, "GET", "/api/v1/identity/me", "", "", oldToken, 200)

	// A partial update does not overwrite other fields; an empty name is allowed.
	body = request(t, app, "PATCH", "/api/v1/identity/admins/"+created.ID, `{"full_name":""}`, "application/json", "owner-token", 200)
	updated := adminResult(t, body)
	if updated.FullName != "" || updated.Email != created.Email || updated.Status != created.Status {
		t.Fatal("partial update changed omitted fields")
	}
	body = request(t, app, "PATCH", "/api/v1/identity/admins/"+created.ID, `{"email":" CHANGED@EXAMPLE.INVALID ","password":"synthetic-new-password","status":"inactive"}`, "application/json", "owner-token", 200)
	updated = adminResult(t, body)
	if updated.Email != "changed@example.invalid" || updated.Status != model.StatusInactive || updated.Role != model.RoleAdmin {
		t.Fatal("wrong updated profile")
	}
	accountLogin(t, app, updated.Email, "synthetic-new-password", 401)
	request(t, app, "GET", "/api/v1/identity/me", "", "", oldToken, 200)
	request(t, app, "PATCH", "/api/v1/identity/admins/"+created.ID, `{"status":"active"}`, "application/json", "owner-token", 200)
	accountLogin(t, app, updated.Email, "synthetic-new-password", 200)
	accountLogin(t, app, updated.Email, "synthetic-old-password", 401)
	accountLogin(t, app, created.Email, "synthetic-new-password", 401)
	request(t, app, "GET", "/api/v1/identity/me", "", "", oldToken, 200)

	// Conflict must leave every requested field untouched.
	before := users.users[id]
	request(t, app, "PATCH", "/api/v1/identity/admins/"+created.ID, `{"email":"customer@example.invalid","full_name":"must not persist","status":"inactive"}`, "application/json", "owner-token", 409)
	if users.users[id] != before {
		t.Fatal("email conflict left a partial update")
	}
	request(t, app, "POST", "/api/v1/identity/admins", `{"email":"customer@example.invalid","password":"synthetic-password"}`, "application/json", "owner-token", 409)
	body = request(t, app, "POST", "/api/v1/identity/admins", `{"email":"inactive@example.invalid","password":"synthetic-password","status":"inactive"}`, "application/json", "owner-token", 201)
	if adminResult(t, body).Status != model.StatusInactive {
		t.Fatal("inactive create not preserved")
	}
	accountLogin(t, app, "inactive@example.invalid", "synthetic-password", 401)
}

func TestAdminMutationDenialAndInput(t *testing.T) {
	app, users, _ := accountTestApp()
	admin, _ := users.FindByEmail(context.Background(), "admin@example.invalid")
	owner, _ := users.FindByEmail(context.Background(), "owner@example.invalid")
	customer, _ := users.FindByEmail(context.Background(), "customer@example.invalid")
	for _, bearer := range []string{"", "bad-token", "customer-token", "admin-token"} {
		status := 403
		if bearer == "" || bearer == "bad-token" {
			status = 401
		}
		request(t, app, "POST", "/api/v1/identity/admins", `{"email":"new@example.invalid","password":"synthetic-password"}`, "application/json", bearer, status)
		request(t, app, "PATCH", "/api/v1/identity/admins/"+admin.ID.String(), `{"status":"inactive"}`, "application/json", bearer, status)
	}
	if users.writes != 0 {
		t.Fatal("unauthorized caller reached mutation")
	}
	for index, tt := range []struct{ method, path, body string }{
		{"POST", "", `null`}, {"POST", "", `[]`}, {"POST", "", `{"role":"OWNER"}`},
		{"POST", "", `{"EMAIL":"new@example.invalid","password":"synthetic-password"}`},
		{"POST", "", `{"email":"new@example.invalid","password":"synthetic-password","status":null}`},
		{"POST", "", `{"email":"invalid","password":"synthetic-password"}`},
		{"POST", "", `{"email":"new@example.invalid","password":"éééé"}`},
		{"POST", "", `{"email":"new@example.invalid","password":"synthetic-password","status":"disabled"}`},
		{"POST", "", `{"email":"new@example.invalid","password":"synthetic-password","status":""}`},
		{"PATCH", "/" + admin.ID.String(), `{}`}, {"PATCH", "/" + admin.ID.String(), `{"email":null}`},
		{"PATCH", "/" + admin.ID.String(), `{"email":""}`}, {"PATCH", "/" + admin.ID.String(), `{"password":""}`},
		{"PATCH", "/" + admin.ID.String(), `{"status":""}`}, {"PATCH", "/" + admin.ID.String(), `{"id":"forged"}`},
		{"PATCH", "/" + admin.ID.String(), `{"password_hash":"forged"}`}, {"PATCH", "/" + admin.ID.String(), `{"role":"CUSTOMER"}`},
		{"PATCH", "/" + admin.ID.String(), `{"updated_at":"forged"}`}, {"PATCH", "/" + admin.ID.String(), `{"unknown":"field"}`},
		{"PATCH", "/invalid", `{"status":"active"}`}, {"PATCH", "/" + admin.ID.String(), `{"full_name":"x"} {}`},
	} {
		t.Run(fmt.Sprintf("%s-case-%d", tt.method, index), func(t *testing.T) {
			request(t, app, tt.method, "/api/v1/identity/admins"+tt.path, tt.body, "application/json", "owner-token", 400)
		})
	}
	request(t, app, "POST", "/api/v1/identity/admins", `{}`, "text/plain", "owner-token", 415)
	request(t, app, "PATCH", "/api/v1/identity/admins/"+admin.ID.String(), `{}`, "text/plain", "owner-token", 415)
	if users.writes != 0 {
		t.Fatal("invalid input reached mutation")
	}
	for _, id := range []uuid.UUID{owner.ID, customer.ID, uuid.New()} {
		request(t, app, "PATCH", "/api/v1/identity/admins/"+id.String(), `{"full_name":"forbidden target"}`, "application/json", "owner-token", 404)
	}
	for _, tt := range []struct {
		err    error
		status int
		code   string
	}{
		{gorm.ErrDuplicatedKey, 409, "EMAIL_ALREADY_REGISTERED"},
		{errors.New("private-database-error"), 500, "INTERNAL_ERROR"},
		{context.DeadlineExceeded, 504, "REQUEST_TIMEOUT"},
	} {
		users.writeErr = tt.err
		for _, method := range []string{"POST", "PATCH"} {
			path := "/api/v1/identity/admins"
			if method == "PATCH" {
				path += "/" + admin.ID.String()
			}
			body := request(t, app, method, path, `{"email":"new@example.invalid","password":"synthetic-password"}`, "application/json", "owner-token", tt.status)
			assertError(t, body, tt.code)
			if strings.Contains(string(body), "private-database") {
				t.Fatal("raw database error exposed")
			}
		}
	}
}
