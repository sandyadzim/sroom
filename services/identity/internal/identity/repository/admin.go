package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"identity/internal/identity/model"
)

func (r *UserRepository) ListAdmins(ctx context.Context, limit, offset int, search, status string) ([]model.User, int64, error) {
	return r.listUsersByRole(ctx, model.RoleAdmin, limit, offset, search, status)
}

func (r *UserRepository) UpdateAdmin(ctx context.Context, id uuid.UUID, changes map[string]any) (*model.User, error) {
	var user model.User
	result := r.db.WithContext(ctx).Model(&user).
		Where("id = ? AND role = ?", id, model.RoleAdmin).
		Select("full_name", "email", "password_hash", "status", "updated_at").
		Clauses(clause.Returning{}).Updates(changes)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &user, nil
}

func (r *UserRepository) FindAdminByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return r.findUserByRole(ctx, id, model.RoleAdmin)
}
