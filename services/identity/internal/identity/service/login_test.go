package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"identity/internal/identity/dto"
	"identity/internal/identity/model"
	"identity/internal/identity/password"
	"identity/internal/identity/token"
)

type fakeLoginUserRepository struct {
	user    *model.User
	findErr error
}

func (f *fakeLoginUserRepository) FindByEmail(
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

type fakeLoginSessionRepository struct {
	session   *model.Session
	createErr error
}

func (f *fakeLoginSessionRepository) Create(
	ctx context.Context,
	session *model.Session,
) error {
	if f.createErr != nil {
		return f.createErr
	}

	f.session = session
	return nil
}

func TestLoginSuccess(t *testing.T) {
	hashedPassword, err := password.Hash("Rahasia123!")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	userID, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("failed to generate user ID: %v", err)
	}

	userRepo := &fakeLoginUserRepository{
		user: &model.User{
			ID:           userID,
			Email:        "budi@example.com",
			PasswordHash: hashedPassword,
		},
	}

	sessionRepo := &fakeLoginSessionRepository{}
	sessionTTL := 7 * 24 * time.Hour
	svc := NewLoginService(userRepo, sessionRepo)

	beforeLogin := time.Now().UTC()

	result, err := svc.Login(
		context.Background(),
		dto.LoginRequest{
			Email:    "BUDI@EXAMPLE.COM",
			Password: "Rahasia123!",
		},
	)

	if err != nil {
		t.Fatalf("Login() failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected login response")
	}

	if result.Token == "" {
		t.Fatal("token must not be empty")
	}

	if result.TokenType != "Bearer" {
		t.Errorf("expected Bearer, got %s", result.TokenType)
	}

	if sessionRepo.session == nil {
		t.Fatal("session was not created")
	}

	if sessionRepo.session.ID.Version() != 7 {
		t.Error("expected session UUID v7")
	}

	if sessionRepo.session.UserID != userID {
		t.Error("session user ID does not match")
	}

	if sessionRepo.session.TokenHash != token.Hash(result.Token) {
		t.Error("stored token hash does not match")
	}

	if sessionRepo.session.TokenHash == result.Token {
		t.Error("plain token must not be stored")
	}

	expectedExpiry := beforeLogin.Add(sessionTTL)

	if sessionRepo.session.ExpiresAt.Before(expectedExpiry) {
		t.Error("session expires earlier than expected")
	}

	if sessionRepo.session.ExpiresAt.After(
		time.Now().UTC().Add(sessionTTL),
	) {
		t.Error("session expires later than expected")
	}

	responseExpiry, err := time.Parse(
		time.RFC3339,
		result.ExpiresAt,
	)
	if err != nil {
		t.Fatalf("invalid response expiry: %v", err)
	}

	if responseExpiry.Unix() != sessionRepo.session.ExpiresAt.Unix() {
		t.Error("response expiry does not match stored session")
	}
}

func TestLoginEmailNotFound(t *testing.T) {
	userRepo := &fakeLoginUserRepository{}
	sessionRepo := &fakeLoginSessionRepository{}
	svc := NewLoginService(userRepo, sessionRepo)

	result, err := svc.Login(
		context.Background(),
		dto.LoginRequest{
			Email:    "unknown@example.com",
			Password: "Rahasia123!",
		},
	)

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}

	if result != nil {
		t.Fatal("expected nil response")
	}

	if sessionRepo.session != nil {
		t.Fatal("session must not be created")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	hashedPassword, err := password.Hash("Rahasia123!")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	userRepo := &fakeLoginUserRepository{
		user: &model.User{
			Email:        "budi@example.com",
			PasswordHash: hashedPassword,
		},
	}

	sessionRepo := &fakeLoginSessionRepository{}
	svc := NewLoginService(userRepo, sessionRepo)

	result, err := svc.Login(
		context.Background(),
		dto.LoginRequest{
			Email:    "budi@example.com",
			Password: "PasswordSalah!",
		},
	)

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}

	if result != nil {
		t.Fatal("expected nil response")
	}

	if sessionRepo.session != nil {
		t.Fatal("session must not be created")
	}
}

func TestLoginDatabaseError(t *testing.T) {
	dbErr := errors.New("database unavailable")

	userRepo := &fakeLoginUserRepository{
		findErr: dbErr,
	}

	sessionRepo := &fakeLoginSessionRepository{}
	svc := NewLoginService(userRepo, sessionRepo)

	result, err := svc.Login(
		context.Background(),
		dto.LoginRequest{
			Email:    "budi@example.com",
			Password: "Rahasia123!",
		},
	)

	if !errors.Is(err, dbErr) {
		t.Fatalf("expected database error, got %v", err)
	}

	if result != nil {
		t.Fatal("expected nil response")
	}

	if sessionRepo.session != nil {
		t.Fatal("session must not be created")
	}
}

func TestLoginSessionCreateError(t *testing.T) {
	hashedPassword, err := password.Hash("Rahasia123!")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	dbErr := errors.New("failed to insert session")

	userRepo := &fakeLoginUserRepository{
		user: &model.User{
			Email:        "budi@example.com",
			PasswordHash: hashedPassword,
		},
	}

	sessionRepo := &fakeLoginSessionRepository{
		createErr: dbErr,
	}

	svc := NewLoginService(userRepo, sessionRepo)

	result, err := svc.Login(
		context.Background(),
		dto.LoginRequest{
			Email:    "budi@example.com",
			Password: "Rahasia123!",
		},
	)

	if !errors.Is(err, dbErr) {
		t.Fatalf("expected session database error, got %v", err)
	}

	if result != nil {
		t.Fatal("token must not be returned when session creation fails")
	}

	if sessionRepo.session != nil {
		t.Fatal("session must not be saved")
	}
}
