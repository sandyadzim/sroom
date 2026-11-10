package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type fakeLogoutSessionRepository struct {
	deletedSessionID uuid.UUID
	deleteErr        error
	deleteCallCount  int
}

func (f *fakeLogoutSessionRepository) DeleteByID(
	ctx context.Context,
	sessionID uuid.UUID,
) error {
	f.deleteCallCount++
	f.deletedSessionID = sessionID

	return f.deleteErr
}

func TestLogoutServiceSuccess(t *testing.T) {
	sessionID := uuid.New()

	repo := &fakeLogoutSessionRepository{}
	logoutService := NewLogoutService(repo)

	err := logoutService.Logout(
		context.Background(),
		sessionID,
	)

	if err != nil {
		t.Fatalf("Logout() returned unexpected error: %v", err)
	}

	if repo.deletedSessionID != sessionID {
		t.Errorf(
			"deleted session ID = %s, want %s",
			repo.deletedSessionID,
			sessionID,
		)
	}

	if repo.deleteCallCount != 1 {
		t.Errorf(
			"DeleteByID() called %d times, want 1",
			repo.deleteCallCount,
		)
	}
}

func TestLogoutServiceSessionAlreadyDeleted(t *testing.T) {
	sessionID := uuid.New()

	// GORM Delete() returns nil error even when
	// no database rows match the session ID.
	repo := &fakeLogoutSessionRepository{
		deleteErr: nil,
	}

	logoutService := NewLogoutService(repo)

	err := logoutService.Logout(
		context.Background(),
		sessionID,
	)

	if err != nil {
		t.Fatalf(
			"Logout() should succeed for an already deleted session: %v",
			err,
		)
	}

	if repo.deletedSessionID != sessionID {
		t.Errorf(
			"deleted session ID = %s, want %s",
			repo.deletedSessionID,
			sessionID,
		)
	}

	if repo.deleteCallCount != 1 {
		t.Errorf(
			"DeleteByID() called %d times, want 1",
			repo.deleteCallCount,
		)
	}
}

func TestLogoutServiceDatabaseError(t *testing.T) {
	sessionID := uuid.New()
	databaseErr := errors.New("database connection failed")

	repo := &fakeLogoutSessionRepository{
		deleteErr: databaseErr,
	}

	logoutService := NewLogoutService(repo)

	err := logoutService.Logout(
		context.Background(),
		sessionID,
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

	if !strings.Contains(err.Error(), "delete session") {
		t.Errorf(
			"error = %q, want context 'delete session'",
			err.Error(),
		)
	}

	if repo.deletedSessionID != sessionID {
		t.Errorf(
			"deleted session ID = %s, want %s",
			repo.deletedSessionID,
			sessionID,
		)
	}

	if repo.deleteCallCount != 1 {
		t.Errorf(
			"DeleteByID() called %d times, want 1",
			repo.deleteCallCount,
		)
	}
}
