package repository

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"identity/internal/identity/model"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

func (r *UserRepository) Create(
	ctx context.Context,
	user *model.User,
) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepository) FindByEmail(
	ctx context.Context,
	email string,
) (*model.User, error) {
	var user model.User

	err := r.db.WithContext(ctx).
		Where("LOWER(email) = LOWER(?)", email).
		First(&user).Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) FindByID(
	ctx context.Context,
	id uuid.UUID,
) (*model.User, error) {
	var user model.User

	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&user).Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error {
	result := r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ? AND role IN ?", id, []model.Role{model.RoleOwner, model.RoleAdmin}).
		Update("password_hash", hash)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *UserRepository) listUsersByRole(ctx context.Context, role model.Role, limit, offset int, search, status string) ([]model.User, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.User{}).Where("role = ?", role)
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

func (r *UserRepository) findUserByRole(ctx context.Context, id uuid.UUID, role model.Role) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).
		Where("id = ? AND role = ?", id, role).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
