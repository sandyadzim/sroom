package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"identity/internal/identity/dto"
	"identity/internal/identity/model"
	"identity/internal/identity/password"
)

type fakeUserRepository struct {
	user      *model.User
	createErr error
	findErr   error
}

func (f *fakeUserRepository) Create(
	ctx context.Context,
	user *model.User,
) error {
	if f.createErr != nil {
		return f.createErr
	}

	f.user = user
	return nil
}

func (f *fakeUserRepository) FindByEmail(
	ctx context.Context,
	email string,
) (*model.User, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}

	if f.user != nil && f.user.Email == email {
		return f.user, nil
	}

	return nil, gorm.ErrRecordNotFound
}

func TestRegisterSuccess(t *testing.T) {
	repo := &fakeUserRepository{}
	svc := NewRegisterService(repo)

	result, err := svc.Register(
		context.Background(),
		dto.RegisterRequest{
			FullName: "Budi Santoso",
			Email:    "BUDI@Example.com",
			Password: "Rahasia123!",
		},
	)

	if repo.user.Role != model.RoleCustomer {
		t.Errorf(
			"user role = %q, want %q",
			repo.user.Role,
			model.RoleCustomer,
		)
	}

	if err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	if result.Email != "budi@example.com" {
		t.Errorf("unexpected email: %s", result.Email)
	}

	if result.FullName != "Budi Santoso" {
		t.Errorf("unexpected full name: %s", result.FullName)
	}

	if _, err := uuid.Parse(result.ID); err != nil {
		t.Fatalf("invalid UUID: %v", err)
	}

	if repo.user.ID.Version() != 7 {
		t.Errorf("expected UUID v7")
	}

	if repo.user.PasswordHash == "Rahasia123!" {
		t.Fatal("password must be hashed")
	}

	valid, err := password.Verify(
		"Rahasia123!",
		repo.user.PasswordHash,
	)
	if err != nil || !valid {
		t.Fatal("stored password hash is invalid")
	}
}

func TestRegisterEmailAlreadyExists(t *testing.T) {
	repo := &fakeUserRepository{
		user: &model.User{
			Email: "budi@example.com",
		},
	}

	svc := NewRegisterService(repo)

	result, err := svc.Register(
		context.Background(),
		dto.RegisterRequest{
			Email:    "budi@example.com",
			Password: "Rahasia123!",
		},
	)

	if !errors.Is(err, ErrEmailAlreadyExists) {
		t.Fatalf("expected ErrEmailAlreadyExists, got %v", err)
	}

	if result != nil {
		t.Fatal("expected nil result")
	}
}

func TestRegisterDatabaseError(t *testing.T) {
	dbErr := errors.New("database unavailable")

	repo := &fakeUserRepository{
		findErr: dbErr,
	}

	svc := NewRegisterService(repo)

	result, err := svc.Register(
		context.Background(),
		dto.RegisterRequest{
			Email:    "budi@example.com",
			Password: "Rahasia123!",
		},
	)

	if !errors.Is(err, dbErr) {
		t.Fatalf("expected database error, got %v", err)
	}

	if result != nil {
		t.Fatal("expected nil result")
	}
}
