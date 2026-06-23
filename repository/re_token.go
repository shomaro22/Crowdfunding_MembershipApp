package repository

import (
	"context"
	"errors"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReTokenRepo interface {
	Create(ctx context.Context, row *entity.ReToken) error
	FindValidByHash(ctx context.Context, hash string, now time.Time) (*entity.ReToken, error)
	Revoke(ctx context.Context, id uuid.UUID, at time.Time) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID, at time.Time) error
}

type reTokenRepo struct {
	db *gorm.DB
}

func NewReTokenRepo(db *gorm.DB) ReTokenRepo {
	return &reTokenRepo{db: db}
}

func (r *reTokenRepo) Create(ctx context.Context, row *entity.ReToken) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *reTokenRepo) FindValidByHash(ctx context.Context, hash string, now time.Time) (*entity.ReToken, error) {
	var (
		m   entity.ReToken
		err error
	)
	err = r.db.WithContext(ctx).
		Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", hash, now).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *reTokenRepo) Revoke(ctx context.Context, id uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).Model(&entity.ReToken{}).
		Where("id = ?", id).
		Update("revoked_at", at).Error
}

func (r *reTokenRepo) RevokeAllForUser(ctx context.Context, userID uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).Model(&entity.ReToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", at).Error
}
