package repository

import (
	"context"

	"gorm.io/gorm"

	"identity/internal/identity/model"
)

type SessionRepository struct {
    db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) *SessionRepository {
    return &SessionRepository{
        db: db,
    }
}

func (r *SessionRepository) Create(
    ctx context.Context,
    session *model.Session,
) error {
    return r.db.WithContext(ctx).Create(session).Error
}
