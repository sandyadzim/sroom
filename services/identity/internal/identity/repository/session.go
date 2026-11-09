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

func (r *SessionRepository) FindByTokenHash(
    ctx context.Context,
    tokenHash string,
) (*model.Session, error) {
    var session model.Session

    err := r.db.WithContext(ctx).
        Where("token_hash = ? AND expires_at > ?", tokenHash, gorm.Expr("NOW()")).
        First(&session).Error

    if err != nil {
        return nil, err
    }

    return &session, nil
}
