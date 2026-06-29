package repository

import (
	"context"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RefreshTokenRepo interface {
	Create(ctx context.Context, row *entity.RefreshToken) error
	FindValidHash(ctx context.Context, hash string, now time.Time) (*entity.RefreshToken, error)
	Revoke(ctx context.Context, id uuid.UUID, at time.Time) error
	RevokeAllByUser(ctx context.Context, userID uuid.UUID, at time.Time) error
}

type refreshTokenRepo struct {
	db *gorm.DB
}

func NewRefreshTokenRepo(db *gorm.DB) RefreshTokenRepo {
	return &refreshTokenRepo{db: db}
}

func (r *refreshTokenRepo) Create(ctx context.Context, row *entity.RefreshToken) error {
	return dbFromCtx(ctx, r.db).Create(row).Error
}

func (r *refreshTokenRepo) FindValidHash(ctx context.Context, hash string, now time.Time) (*entity.RefreshToken, error) {
	return firstOnePtr[entity.RefreshToken](dbFromCtx(ctx, r.db).
		Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", hash, now))
}

func (r *refreshTokenRepo) Revoke(ctx context.Context, id uuid.UUID, at time.Time) error {
	return dbFromCtx(ctx, r.db).Model(&entity.RefreshToken{}).
		Where("id = ?", id).
		Update("revoked_at", at).Error
}

func (r *refreshTokenRepo) RevokeAllByUser(ctx context.Context, userID uuid.UUID, at time.Time) error {
	return dbFromCtx(ctx, r.db).Model(&entity.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", at).Error
}
