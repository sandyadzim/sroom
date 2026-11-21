package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"identity/internal/identity/model"
)

func (r *UserRepository) ListCustomers(ctx context.Context, limit, offset int, search, status string) ([]model.User, int64, error) {
	return r.listUsersByRole(ctx, model.RoleCustomer, limit, offset, search, status)
}

func (r *UserRepository) FindCustomerByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return r.findUserByRole(ctx, id, model.RoleCustomer)
}

func (r *UserRepository) FindCustomersByIDs(ctx context.Context, ids []uuid.UUID) ([]model.User, error) {
	if len(ids) == 0 {
		return []model.User{}, nil
	}
	var users []model.User
	err := r.db.WithContext(ctx).
		Where("id IN ? AND role = ?", ids, model.RoleCustomer).
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepository) UpdateCustomerStatus(ctx context.Context, id uuid.UUID, status string) (*model.User, error) {
	var user model.User
	result := r.db.WithContext(ctx).Model(&user).
		Where("id = ? AND role = ?", id, model.RoleCustomer).
		Clauses(clause.Returning{}).Update("status", status)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &user, nil
}
