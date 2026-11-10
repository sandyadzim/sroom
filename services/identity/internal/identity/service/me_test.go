package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"identity/internal/identity/model"
)

type fakeMeUserRepository struct {
	user          *model.User
	findErr       error
	receivedID    uuid.UUID
	findCallCount int
}

func (f *fakeMeUserRepository) FindByID(
	ctx context.Context,
	id uuid.UUID,
) (*model.User, error) {
	f.findCallCount++
	f.receivedID = id

	if f.findErr != nil {
		return nil, f.findErr
	}

	return f.user, nil
}

func TestMeServiceGetProfileSuccess(t *testing.T) {
	userID := uuid.New()
	createdAt := time.Date(
		2026, time.November, 9,
		10, 30, 0, 0,
		time.FixedZone("WIB", 7*60*60),
	)

	expectedUser := &model.User{
		ID:           userID,
		FullName:     "Budi Santoso",
		Email:        "budi@example.com",
		PasswordHash: "secret-password-hash",
		Role:         model.RoleCustomer,
		CreatedAt:    createdAt,
	}

	repo := &fakeMeUserRepository{
		user: expectedUser,
	}

	meService := NewMeService(repo)

	result, err := meService.GetProfile(
		context.Background(),
		userID,
	)

	if result.Role != model.RoleCustomer {
		t.Errorf(
			"Role = %q, want %q",
			result.Role,
			model.RoleCustomer,
		)
	}

	if err != nil {
		t.Fatalf("GetProfile() returned unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("GetProfile() returned nil response")
	}

	if result.ID != userID.String() {
		t.Errorf(
			"ID = %q, want %q",
			result.ID,
			userID.String(),
		)
	}

	if result.FullName != expectedUser.FullName {
		t.Errorf(
			"FullName = %q, want %q",
			result.FullName,
			expectedUser.FullName,
		)
	}

	if result.Email != expectedUser.Email {
		t.Errorf(
			"Email = %q, want %q",
			result.Email,
			expectedUser.Email,
		)
	}

	expectedCreatedAt := "2026-11-09T03:30:00Z"

	if result.CreatedAt != expectedCreatedAt {
		t.Errorf(
			"CreatedAt = %q, want %q",
			result.CreatedAt,
			expectedCreatedAt,
		)
	}

	if repo.receivedID != userID {
		t.Errorf(
			"repository received ID = %s, want %s",
			repo.receivedID,
			userID,
		)
	}

	if repo.findCallCount != 1 {
		t.Errorf(
			"repository called %d times, want 1",
			repo.findCallCount,
		)
	}

	// Verify that the JSON response does not expose
	// the password hash.
	responseJSON, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal() returned error: %v", err)
	}

	if strings.Contains(
		string(responseJSON),
		expectedUser.PasswordHash,
	) {
		t.Error("response JSON must not contain password hash")
	}

	if strings.Contains(
		string(responseJSON),
		"password_hash",
	) {
		t.Error("response JSON must not contain password_hash field")
	}
}

func TestMeServiceGetProfileUserNotFound(t *testing.T) {
	userID := uuid.New()

	repo := &fakeMeUserRepository{
		findErr: gorm.ErrRecordNotFound,
	}

	meService := NewMeService(repo)

	result, err := meService.GetProfile(
		context.Background(),
		userID,
	)

	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf(
			"error = %v, want ErrUserNotFound",
			err,
		)
	}

	if result != nil {
		t.Error("expected nil response")
	}

	if repo.receivedID != userID {
		t.Errorf(
			"repository received ID = %s, want %s",
			repo.receivedID,
			userID,
		)
	}

	if repo.findCallCount != 1 {
		t.Errorf(
			"repository called %d times, want 1",
			repo.findCallCount,
		)
	}
}

func TestMeServiceGetProfileDatabaseError(t *testing.T) {
	userID := uuid.New()
	databaseErr := errors.New("database connection failed")

	repo := &fakeMeUserRepository{
		findErr: databaseErr,
	}

	meService := NewMeService(repo)

	result, err := meService.GetProfile(
		context.Background(),
		userID,
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

	if errors.Is(err, ErrUserNotFound) {
		t.Error("database error must not become ErrUserNotFound")
	}

	if !strings.Contains(err.Error(), "find user by ID") {
		t.Errorf(
			"error = %q, want context 'find user by ID'",
			err.Error(),
		)
	}

	if result != nil {
		t.Error("expected nil response")
	}

	if repo.receivedID != userID {
		t.Errorf(
			"repository received ID = %s, want %s",
			repo.receivedID,
			userID,
		)
	}

	if repo.findCallCount != 1 {
		t.Errorf(
			"repository called %d times, want 1",
			repo.findCallCount,
		)
	}
}
