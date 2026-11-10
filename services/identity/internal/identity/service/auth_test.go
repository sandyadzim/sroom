package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"identity/internal/identity/model"
	"identity/internal/identity/token"
)

type fakeAuthSessionRepository struct {
	session       *model.Session
	findErr       error
	receivedHash  string
	findCallCount int
}

func (f *fakeAuthSessionRepository) FindByTokenHash(
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

func TestAuthenticateSuccess(t *testing.T) {
	plainToken := "valid-opaque-token-for-testing"

	userID := uuid.New()
	sessionID := uuid.New()

	expectedSession := &model.Session{
		ID:        sessionID,
		UserID:    userID,
		TokenHash: token.Hash(plainToken),
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}

	repo := &fakeAuthSessionRepository{
		session: expectedSession,
	}

	authService := NewAuthService(repo)

	session, err := authService.Authenticate(
		context.Background(),
		plainToken,
	)
	if err != nil {
		t.Fatalf("Authenticate() returned unexpected error: %v", err)
	}

	if session == nil {
		t.Fatal("Authenticate() returned nil session")
	}

	if session.ID != expectedSession.ID {
		t.Errorf(
			"session ID = %s, want %s",
			session.ID,
			expectedSession.ID,
		)
	}

	if session.UserID != expectedSession.UserID {
		t.Errorf(
			"user ID = %s, want %s",
			session.UserID,
			expectedSession.UserID,
		)
	}

	expectedHash := token.Hash(plainToken)

	if repo.receivedHash != expectedHash {
		t.Errorf(
			"repository received hash = %q, want %q",
			repo.receivedHash,
			expectedHash,
		)
	}

	if repo.receivedHash == plainToken {
		t.Error("repository received plain token instead of token hash")
	}

	if repo.findCallCount != 1 {
		t.Errorf(
			"repository called %d times, want 1",
			repo.findCallCount,
		)
	}
}

func TestAuthenticateEmptyToken(t *testing.T) {
	repo := &fakeAuthSessionRepository{}

	authService := NewAuthService(repo)

	session, err := authService.Authenticate(
		context.Background(),
		"",
	)

	if !errors.Is(err, ErrInvalidSession) {
		t.Errorf(
			"error = %v, want ErrInvalidSession",
			err,
		)
	}

	if session != nil {
		t.Error("expected nil session")
	}

	if repo.findCallCount != 0 {
		t.Errorf(
			"repository called %d times, want 0",
			repo.findCallCount,
		)
	}
}

func TestAuthenticateTokenNotFound(t *testing.T) {
	repo := &fakeAuthSessionRepository{
		findErr: gorm.ErrRecordNotFound,
	}

	authService := NewAuthService(repo)

	session, err := authService.Authenticate(
		context.Background(),
		"unknown-token",
	)

	if !errors.Is(err, ErrInvalidSession) {
		t.Errorf(
			"error = %v, want ErrInvalidSession",
			err,
		)
	}

	if session != nil {
		t.Error("expected nil session")
	}

	if repo.receivedHash != token.Hash("unknown-token") {
		t.Error("repository did not receive the expected token hash")
	}

	if repo.findCallCount != 1 {
		t.Errorf(
			"repository called %d times, want 1",
			repo.findCallCount,
		)
	}
}

func TestAuthenticateExpiredSession(t *testing.T) {
	// FindByTokenHash filters sessions using:
	// expires_at > NOW()
	//
	// An expired session is therefore reported by GORM
	// as ErrRecordNotFound.

	repo := &fakeAuthSessionRepository{
		findErr: gorm.ErrRecordNotFound,
	}

	authService := NewAuthService(repo)

	session, err := authService.Authenticate(
		context.Background(),
		"expired-token",
	)

	if !errors.Is(err, ErrInvalidSession) {
		t.Errorf(
			"error = %v, want ErrInvalidSession",
			err,
		)
	}

	if session != nil {
		t.Error("expected nil session for expired token")
	}

	if repo.findCallCount != 1 {
		t.Errorf(
			"repository called %d times, want 1",
			repo.findCallCount,
		)
	}
}

func TestAuthenticateDatabaseError(t *testing.T) {
	databaseErr := errors.New("database connection failed")

	repo := &fakeAuthSessionRepository{
		findErr: databaseErr,
	}

	authService := NewAuthService(repo)

	session, err := authService.Authenticate(
		context.Background(),
		"valid-looking-token",
	)

	if err == nil {
		t.Fatal("expected database error, got nil")
	}

	if !errors.Is(err, databaseErr) {
		t.Errorf(
			"error = %v, want wrapped database error",
			err,
		)
	}

	if errors.Is(err, ErrInvalidSession) {
		t.Error("database error must not become ErrInvalidSession")
	}

	if !strings.Contains(err.Error(), "find session") {
		t.Errorf(
			"error = %q, want context 'find session'",
			err.Error(),
		)
	}

	if session != nil {
		t.Error("expected nil session")
	}

	if repo.findCallCount != 1 {
		t.Errorf(
			"repository called %d times, want 1",
			repo.findCallCount,
		)
	}
}
