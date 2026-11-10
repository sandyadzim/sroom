package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"identity/internal/identity/middleware"
	"identity/internal/identity/model"
	"identity/internal/identity/password"
	"identity/internal/identity/response"
	"identity/internal/identity/service"
)

func newTestApp(users *testUsers) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: response.ErrorHandler,
		BodyLimit:    middleware.MaxBodyBytes,
	})
	middleware.UseRequests(app, middleware.DefaultRequestTimeout, slog.New(slog.NewTextHandler(io.Discard, nil)))
	sessions := &testSessions{}
	auth := middleware.NewAuthMiddleware(service.NewAuthService(sessions))
	app.Post("/api/v1/identity/register", NewRegisterHandler(service.NewRegisterService(users)).Register)
	app.Post("/api/v1/identity/login", NewLoginHandler(service.NewLoginService(users, sessions)).Login)
	app.Get("/api/v1/identity/me", auth.RequireAuth, NewMeHandler(service.NewMeService(users)).Me)
	app.Post("/api/v1/identity/logout", auth.RequireAuth, NewLogoutHandler(service.NewLogoutService(sessions)).Logout)
	return app
}

func request(t *testing.T, app *fiber.App, method, path, body, contentType, token string, wantStatus int) []byte {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	// Real Argon2id hashing takes longer under the race detector and parallel package tests.
	res, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second, FailOnTimeout: true})
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != wantStatus {
		t.Fatalf("status = %d, want %d", res.StatusCode, wantStatus)
	}
	requestID := res.Header.Get("X-Request-ID")
	if requestID == "" || res.Header.Get("Cache-Control") != "no-store" || res.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("request ID, private cache policy, and security headers must be present")
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if wantStatus != 204 && !strings.HasPrefix(res.Header.Get("Content-Type"), "application/json") {
		t.Fatal("expected JSON content type")
	}
	if wantStatus >= 400 {
		var result response.ErrorResponse
		if err := json.Unmarshal(data, &result); err != nil || result.RequestID != requestID {
			t.Fatal("error request_id must match the response header")
		}
	}
	return data
}

func TestAuthSuccessResponses(t *testing.T) {
	users := &testUsers{}
	app := newTestApp(users)
	body := request(t, app, "POST", "/api/v1/identity/register",
		`{"email":"synthetic@example.invalid","password":"synthetic-password","role":"OWNER","user_id":"forged"}`,
		"application/json; charset=utf-8", "", 201)
	var registered struct {
		Data struct {
			ID       string `json:"id"`
			FullName string `json:"full_name"`
			Email    string `json:"email"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &registered); err != nil {
		t.Fatal(err)
	}
	if registered.Data.ID != users.user.ID.String() || registered.Data.Email != "synthetic@example.invalid" {
		t.Fatal("register response must contain the created user under data")
	}
	if users.user.Role != model.RoleCustomer || registered.Data.FullName != "" {
		t.Fatal("registration must preserve CUSTOMER role and optional name")
	}
	if strings.Contains(string(body), "password") || strings.Contains(string(body), "token") {
		t.Fatal("register response must not contain credentials")
	}

	body = request(t, app, "POST", "/api/v1/identity/login",
		`{"email":"synthetic@example.invalid","password":"synthetic-password"}`, "application/json", "", 200)
	var login struct {
		Data struct {
			Token     string `json:"token"`
			TokenType string `json:"token_type"`
			ExpiresAt string `json:"expires_at"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &login); err != nil {
		t.Fatal(err)
	}
	if login.Data.Token == "" || login.Data.TokenType != "Bearer" {
		t.Fatal("login must return a Bearer token under data")
	}
	if _, err := time.Parse(time.RFC3339, login.Data.ExpiresAt); err != nil {
		t.Fatal("login expiry must be RFC3339")
	}

	body = request(t, app, "GET", "/api/v1/identity/me", "", "", login.Data.Token, 200)
	var profile struct {
		Data struct {
			ID   string     `json:"id"`
			Role model.Role `json:"role"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &profile); err != nil {
		t.Fatal(err)
	}
	if profile.Data.ID != registered.Data.ID || profile.Data.Role != model.RoleCustomer {
		t.Fatal("me must return the authenticated user under data")
	}
	if strings.Contains(string(body), "password") || strings.Contains(string(body), "token") {
		t.Fatal("profile must not contain credentials")
	}
	body = request(t, app, "POST", "/api/v1/identity/logout", "", "", login.Data.Token, 204)
	if len(body) != 0 {
		t.Fatal("logout 204 must have an empty body")
	}
	assertError(t, request(t, app, "GET", "/api/v1/identity/me", "", "", login.Data.Token, 401), "UNAUTHENTICATED")
}

func TestAuthErrorResponses(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, body, contentType, code string
		status                                      int
	}{
		{"malformed JSON", "POST", "/api/v1/identity/register", `{`, "application/json", "INVALID_REQUEST", 400},
		{"invalid field", "POST", "/api/v1/identity/register", `{"email":"invalid","password":"short"}`, "application/json", "VALIDATION_ERROR", 400},
		{"password byte limit", "POST", "/api/v1/identity/register", `{"email":"a@example.invalid","password":"` + strings.Repeat("a", 73) + `"}`, "application/json", "VALIDATION_ERROR", 400},
		{"invalid login", "POST", "/api/v1/identity/login", `{}`, "application/json", "VALIDATION_ERROR", 400},
		{"unknown email", "POST", "/api/v1/identity/login", `{"email":"a@example.invalid","password":"synthetic-password"}`, "application/json", "INVALID_CREDENTIALS", 401},
		{"no token", "GET", "/api/v1/identity/me", "", "", "UNAUTHENTICATED", 401},
		{"logout no token", "POST", "/api/v1/identity/logout", "", "", "UNAUTHENTICATED", 401},
		{"unknown route", "GET", "/not-found", "", "", "NOT_FOUND", 404},
		{"wrong method", "PUT", "/api/v1/identity/login", "", "", "METHOD_NOT_ALLOWED", 405},
		{"non-JSON input", "POST", "/api/v1/identity/login", `email=a`, "text/plain", "UNSUPPORTED_MEDIA_TYPE", 415},
		{"missing content type", "POST", "/api/v1/identity/register", `{}`, "", "UNSUPPORTED_MEDIA_TYPE", 415},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := request(t, newTestApp(&testUsers{}), tc.method, tc.path, tc.body, tc.contentType, "", tc.status)
			result := assertError(t, body, tc.code)
			if tc.code == "VALIDATION_ERROR" && len(result.Error.Details) == 0 {
				t.Fatal("field validation must include field errors")
			}
		})
	}
}

func TestDuplicateEmailResponse(t *testing.T) {
	users := &testUsers{user: &model.User{Email: "synthetic@example.invalid"}}
	body := request(t, newTestApp(users), "POST", "/api/v1/identity/register",
		`{"email":"SYNTHETIC@example.invalid","password":"synthetic-password"}`, "application/json", "", 409)
	assertError(t, body, "EMAIL_ALREADY_REGISTERED")
}

func TestInternalErrorDoesNotExposeRepositoryDetails(t *testing.T) {
	users := &testUsers{findErr: errors.New("synthetic-private-database-detail")}
	body := request(t, newTestApp(users), "POST", "/api/v1/identity/login",
		`{"email":"synthetic@example.invalid","password":"synthetic-password"}`, "application/json", "", 500)
	assertError(t, body, "INTERNAL_ERROR")
	if strings.Contains(string(body), "synthetic-private") {
		t.Fatal("response must not expose repository error details")
	}
}

func TestOversizedBodyResponse(t *testing.T) {
	// app.Test returns a transport error for oversized bodies; use real HTTP here.
	users := &testUsers{}
	app := newTestApp(users)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- app.Listener(listener, fiber.ListenConfig{DisableStartupMessage: true})
	}()
	t.Cleanup(func() {
		if err := app.ShutdownWithTimeout(2 * time.Second); err != nil {
			t.Error(err)
		}
		if err := <-serverErrors; err != nil {
			t.Error(err)
		}
	})
	conn, err := net.DialTimeout("tcp", listener.Addr().String(), 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	// The server rejects this Content-Length before reading the body. Sending
	// headers only avoids a client write racing the server's early connection close.
	_, err = fmt.Fprintf(conn, "POST /api/v1/identity/login HTTP/1.1\r\nHost: localhost\r\nContent-Type: application/json\r\nContent-Length: %d\r\nConnection: close\r\n\r\n", middleware.MaxBodyBytes+1)
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.ReadResponse(bufio.NewReader(conn), httptest.NewRequest("POST", "/api/v1/identity/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 413 || !strings.HasPrefix(res.Header.Get("Content-Type"), "application/json") {
		t.Fatal("oversized body must return HTTP 413 JSON")
	}
	if res.Header.Get("X-Request-ID") == "" || res.Header.Get("Cache-Control") != "no-store" || res.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("oversized body error must retain request and security headers")
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	result := assertError(t, body, "PAYLOAD_TOO_LARGE")
	if result.RequestID != res.Header.Get("X-Request-ID") {
		t.Fatal("oversized body error request_id must match header")
	}
	if users.findCalls != 0 || users.user != nil {
		t.Fatal("oversized body must not reach the repository or create a user")
	}
}

func TestRequestDeadlineReachesRepository(t *testing.T) {
	users := &testUsers{waitForDeadline: true}
	app := fiber.New(fiber.Config{ErrorHandler: response.ErrorHandler, BodyLimit: middleware.MaxBodyBytes})
	middleware.UseRequests(app, 20*time.Millisecond, slog.New(slog.NewTextHandler(io.Discard, nil)))
	app.Post("/api/v1/identity/login", NewLoginHandler(service.NewLoginService(users, &testSessions{})).Login)
	started := time.Now()
	body := request(t, app, "POST", "/api/v1/identity/login",
		`{"email":"synthetic@example.invalid","password":"synthetic-password"}`, "application/json", "", 504)
	assertError(t, body, "REQUEST_TIMEOUT")
	if users.receivedContext == nil {
		t.Fatal("repository must receive the standard request context")
	}
	if _, ok := users.receivedContext.Deadline(); !ok || !errors.Is(users.receivedContext.Err(), context.DeadlineExceeded) {
		t.Fatal("repository must observe the request deadline")
	}
	if time.Since(started) > time.Second {
		t.Fatal("cooperating repository must return promptly after deadline")
	}
}

func assertError(t *testing.T, body []byte, code string) response.ErrorResponse {
	t.Helper()
	var result response.ErrorResponse
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatal("expected error envelope")
	}
	if result.Error.Code != code || result.Error.Message == "" || result.Error.Details == nil {
		t.Fatalf("expected error code %s, message, and details object", code)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		t.Fatal(err)
	}
	if _, ok := fields["request_id"]; !ok || result.RequestID == "" {
		t.Fatal("request_id must be present and nonempty")
	}
	return result
}

type testUsers struct {
	user            *model.User
	findErr         error
	findCalls       int
	waitForDeadline bool
	receivedContext context.Context
}

func TestInactiveAdminLoginAndExistingSession(t *testing.T) {
	hash, err := password.Hash("synthetic-password")
	if err != nil {
		t.Fatal(err)
	}
	users := &testUsers{user: &model.User{
		ID: uuid.New(), Email: "admin@example.invalid", Role: model.RoleAdmin,
		Status: model.StatusActive, PasswordHash: hash,
	}}
	app := newTestApp(users)
	loginBody := `{"email":"admin@example.invalid","password":"synthetic-password"}`
	body := request(t, app, "POST", "/api/v1/identity/login", loginBody, "application/json", "", 200)
	var login struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &login); err != nil {
		t.Fatal(err)
	}
	if login.Data.Token == "" {
		t.Fatal("active ADMIN must receive a token")
	}

	users.user.Status = model.StatusInactive
	body = request(t, app, "POST", "/api/v1/identity/login", loginBody, "application/json", "", 401)
	assertError(t, body, "INVALID_CREDENTIALS")
	request(t, app, "GET", "/api/v1/identity/me", "", "", login.Data.Token, 200)
	request(t, app, "POST", "/api/v1/identity/logout", "", "", login.Data.Token, 204)
	request(t, app, "GET", "/api/v1/identity/me", "", "", login.Data.Token, 401)
}

func (r *testUsers) Create(_ context.Context, user *model.User) error {
	r.user = user
	return nil
}
func (r *testUsers) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	r.findCalls++
	r.receivedContext = ctx
	if r.waitForDeadline {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if r.findErr != nil {
		return nil, r.findErr
	}
	if r.user == nil || r.user.Email != email {
		return nil, gorm.ErrRecordNotFound
	}
	return r.user, nil
}
func (r *testUsers) FindByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	if r.user == nil || r.user.ID != id {
		return nil, gorm.ErrRecordNotFound
	}
	return r.user, nil
}

type testSessions struct{ session *model.Session }

func (r *testSessions) Create(_ context.Context, session *model.Session) error {
	r.session = session
	return nil
}
func (r *testSessions) FindByTokenHash(_ context.Context, hash string) (*model.Session, error) {
	if r.session == nil || r.session.TokenHash != hash {
		return nil, gorm.ErrRecordNotFound
	}
	return r.session, nil
}
func (r *testSessions) DeleteByID(_ context.Context, id uuid.UUID) error {
	if r.session != nil && r.session.ID == id {
		r.session = nil
	}
	return nil
}
