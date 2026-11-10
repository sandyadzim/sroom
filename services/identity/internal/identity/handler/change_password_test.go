package handler

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"identity/internal/identity/dto"
	"identity/internal/identity/model"
	"identity/internal/identity/password"
	"identity/internal/identity/token"
)

func TestChangeOwnPasswordLifecycle(t *testing.T) {
	for _, role := range []model.Role{model.RoleOwner, model.RoleAdmin} {
		t.Run(string(role), func(t *testing.T) {
			app, users, sessions := accountTestApp()
			email := strings.ToLower(string(role)) + "@example.invalid"
			user, _ := users.FindByEmail(context.Background(), email)
			hash, err := password.Hash("synthetic-old-password")
			if err != nil {
				t.Fatal(err)
			}
			user.PasswordHash = hash
			users.users[user.ID] = *user
			before := *user
			bearer := strings.ToLower(string(role)) + "-token"
			sessions.sessions[token.Hash("second-session")] = &model.Session{ID: uuid.New(), UserID: user.ID, TokenHash: token.Hash("second-session"), ExpiresAt: time.Now().Add(time.Hour)}
			beforeSessions := len(sessions.sessions)
			wrong := request(t, app, "POST", "/api/v1/identity/change-password", `{"current_password":"wrong-password","new_password":"synthetic-new-password"}`, "application/json", bearer, 400)
			assertError(t, wrong, "INVALID_CURRENT_PASSWORD")
			if users.writes != 0 {
				t.Fatal("wrong current password must not mutate account")
			}
			newPassword := "  éééééééé  " // Spaces and Unicode must survive unchanged.
			payload, err := json.Marshal(dto.ChangePasswordRequest{CurrentPassword: "synthetic-old-password", NewPassword: newPassword})
			if err != nil {
				t.Fatal(err)
			}
			body := request(t, app, "POST", "/api/v1/identity/change-password", string(payload), "application/json", bearer, 204)
			if len(body) != 0 {
				t.Fatal("204 must have an empty body")
			}
			after := users.users[user.ID]
			if after.PasswordHash == hash {
				t.Fatal("password hash did not change")
			}
			if valid, err := password.Verify(newPassword, after.PasswordHash); err != nil || !valid {
				t.Fatal("new password was trimmed or not hashed")
			}
			after.PasswordHash, after.UpdatedAt = before.PasswordHash, before.UpdatedAt
			if after != before {
				t.Fatal("change-password changed other profile fields")
			}
			if len(sessions.sessions) != beforeSessions {
				t.Fatal("password change must preserve all sessions")
			}
			request(t, app, "GET", "/api/v1/identity/me", "", "", bearer, 200)
			request(t, app, "GET", "/api/v1/identity/me", "", "", "second-session", 200)
			accountLogin(t, app, email, "synthetic-old-password", 401)
			accountLogin(t, app, email, newPassword, 200)
		})
	}
}

func TestChangePasswordDenialsAndValidation(t *testing.T) {
	app, users, sessions := accountTestApp()
	validBody := `{"current_password":"synthetic-old-password","new_password":"synthetic-new-password"}`
	for _, tt := range []struct {
		bearer string
		status int
	}{
		{"", 401}, {"bad-token", 401}, {"customer-token", 403},
	} {
		request(t, app, "POST", "/api/v1/identity/change-password", validBody, "application/json", tt.bearer, tt.status)
	}
	for _, body := range []string{
		`{}`, `null`, `[]`, `{"current_password":null,"new_password":"synthetic-password"}`,
		`{"current_password":"x","new_password":null}`,
		`{"current_password":"x","new_password":"éééé"}`,
		`{"current_password":"x","new_password":""}`,
		`{"new_password":"synthetic-password"}`,
		`{"current_password":"x","new_password":"synthetic-password","user_id":"forged"}`,
		`{"current_password":"x","new_password":"synthetic-password","role":"OWNER"}`,
		`{"current_password":"x","new_password":"` + strings.Repeat("é", 37) + `"}`,
	} {
		request(t, app, "POST", "/api/v1/identity/change-password", body, "application/json", "owner-token", 400)
	}
	request(t, app, "POST", "/api/v1/identity/change-password", validBody, "text/plain", "owner-token", 415)
	if users.writes != 0 {
		t.Fatal("denied/invalid password request mutated an account")
	}
	adminID := sessions.sessions[token.Hash("admin-token")].UserID
	delete(users.users, adminID)
	request(t, app, "POST", "/api/v1/identity/change-password", validBody, "application/json", "admin-token", 401)
	sessions.sessions[token.Hash("owner-token")].ExpiresAt = time.Now().Add(-time.Hour)
	request(t, app, "POST", "/api/v1/identity/change-password", validBody, "application/json", "owner-token", 401)
}

func TestChangePasswordRepositoryErrors(t *testing.T) {
	app, users, sessions := accountTestApp()
	id := sessions.sessions[token.Hash("owner-token")].UserID
	owner := users.users[id]
	hash, err := password.Hash("synthetic-old-password")
	if err != nil {
		t.Fatal(err)
	}
	owner.PasswordHash = hash
	users.users[id] = owner
	for _, tt := range []struct {
		err    error
		status int
		code   string
	}{
		{gorm.ErrRecordNotFound, 401, "UNAUTHENTICATED"},
		{errors.New("private-password-update-error"), 500, "INTERNAL_ERROR"},
		{context.DeadlineExceeded, 504, "REQUEST_TIMEOUT"},
	} {
		users.writeErr = tt.err
		body := request(t, app, "POST", "/api/v1/identity/change-password", `{"current_password":"synthetic-old-password","new_password":"synthetic-new-password"}`, "application/json", "owner-token", tt.status)
		assertError(t, body, tt.code)
		if strings.Contains(string(body), "private-password") {
			t.Fatal("raw database error exposed")
		}
		if users.users[id].PasswordHash != hash {
			t.Fatal("failed write changed password")
		}
	}
}
