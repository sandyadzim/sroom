package repository

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"identity/internal/identity/model"
)

func (r *UserRepository) ListAdmins(ctx context.Context, limit, offset int, search, status string) ([]model.User, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.User{}).Where("role = ?", model.RoleAdmin)
	if search != "" {
		// Treat %, _ and backslash as literal characters, not search wildcards.
		escaped := strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(search)
		pattern := "%" + escaped + "%"
		query = query.Where("(full_name ILIKE ? OR email ILIKE ?)", pattern, pattern)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	// Copy the query so counting does not change the query used to read rows.
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var users []model.User
	err := query.
		Order("created_at DESC, id DESC").
		Limit(limit).Offset(offset).
		Find(&users).Error
	if err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (r *UserRepository) FindAdminByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).
		Where("id = ? AND role = ?", id, model.RoleAdmin).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
