package repository

import (
	"context"
	"time"

	"crowdfund-app/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RefreshTokenRepo interface {
	Create(ctx context.Context, row *model.RefreshToken) error
	FindValidHash(ctx context.Context, hash string, now time.Time) (*model.RefreshToken, error)
	Revoke(ctx context.Context, id uuid.UUID, at time.Time) error
	RevokeAllByUser(ctx context.Context, userID uuid.UUID, at time.Time) error
}

type refreshTokenRepo struct {
	db *gorm.DB
}

func NewRefreshTokenRepo(db *gorm.DB) RefreshTokenRepo {
	return &refreshTokenRepo{db: db}
}

func (r *refreshTokenRepo) Create(ctx context.Context, row *model.RefreshToken) error {
	return dbFromCtx(ctx, r.db).Create(row).Error
}

func (r *refreshTokenRepo) FindValidHash(ctx context.Context, hash string, now time.Time) (*model.RefreshToken, error) {
	return firstOnePtr[model.RefreshToken](dbFromCtx(ctx, r.db).
		Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", hash, now))
}

func (r *refreshTokenRepo) Revoke(ctx context.Context, id uuid.UUID, at time.Time) error {
	return dbFromCtx(ctx, r.db).Model(&model.RefreshToken{}).
		Where("id = ?", id).
		Update("revoked_at", at).Error
}

func (r *refreshTokenRepo) RevokeAllByUser(ctx context.Context, userID uuid.UUID, at time.Time) error {
	return dbFromCtx(ctx, r.db).Model(&model.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", at).Error
}
