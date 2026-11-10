package repository

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"identity/internal/identity/model"
)

func TestAccountMutationSQL(t *testing.T) {
	db, err := gorm.Open(postgres.Open("host=synthetic.invalid dbname=synthetic"), &gorm.Config{
		DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true, Logger: logger.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pool.Close(); err != nil {
			t.Error(err)
		}
	})
	var query string
	var args []any
	capture := func(tx *gorm.DB) {
		query = tx.Statement.SQL.String()
		args = append([]any(nil), tx.Statement.Vars...)
	}
	if err := db.Callback().Update().After("gorm:update").Register("test:capture_account_update", capture); err != nil {
		t.Fatal(err)
	}
	if err := db.Callback().Create().After("gorm:create").Register("test:capture_account_create", capture); err != nil {
		t.Fatal(err)
	}
	repo := NewUserRepository(db)
	id := uuid.New()
	_, err = repo.UpdateAdmin(context.Background(), id, map[string]any{
		"full_name": "", "email": "new@example.invalid", "status": model.StatusInactive,
		"password_hash": "synthetic-hash", "role": model.RoleOwner, "id": uuid.New(),
	})
	// DryRun does not execute rows, so the production not-found guard is expected.
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("expected zero-row DryRun result")
	}
	if !strings.Contains(query, `UPDATE "users" SET`) || !strings.Contains(query, "RETURNING *") || !strings.Contains(query, "WHERE id = $") || !strings.Contains(query, "AND role = $") {
		t.Fatal("ADMIN mutation must use one UPDATE with role/ID filtering and returned row")
	}
	setClause := strings.Split(strings.Split(query, " SET ")[1], " WHERE ")[0]
	if strings.Contains(setClause, `"role"`) || strings.Contains(setClause, `"id"`) || !strings.Contains(setClause, `"updated_at"`) || !strings.Contains(setClause, `"full_name"`) {
		t.Fatal("mutation must use an explicit column allowlist and update timestamp/empty name")
	}
	if args[len(args)-2] != id || args[len(args)-1] != model.RoleAdmin {
		t.Fatal("wrong mutation target filter")
	}

	err = repo.UpdatePassword(context.Background(), id, "synthetic-new-hash")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("expected zero-row DryRun result")
	}
	if !strings.Contains(query, `"password_hash"=`) || !strings.Contains(query, "role IN") || !strings.Contains(query, "WHERE id = $") {
		t.Fatal("password update must filter internal role and user ID")
	}
	if strings.Contains(query, "synthetic-new-hash") {
		t.Fatal("hash must stay bound as a SQL parameter")
	}
	if args[len(args)-3] != id || args[len(args)-2] != model.RoleOwner || args[len(args)-1] != model.RoleAdmin {
		t.Fatal("wrong own-password target filter")
	}

	err = repo.Create(context.Background(), &model.User{ID: uuid.New(), Email: "new@example.invalid", PasswordHash: "synthetic-hash", Role: model.RoleAdmin, Status: model.StatusActive})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(query, `INSERT INTO "users"`) || !strings.Contains(query, `"role"`) || !strings.Contains(query, `"status"`) {
		t.Fatal("create must insert ADMIN/status directly")
	}
}
