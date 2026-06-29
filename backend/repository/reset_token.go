package repository

import (
	"context"
	"time"

	"crowdfund-app/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IResetRepo interface {
	Create(ctx context.Context, row *model.ResetToken) error
	GetValidHash(ctx context.Context, hash string, now time.Time) (model.ResetToken, error)
	MarkUsed(ctx context.Context, id uuid.UUID, at time.Time) error
}

type resetRepo struct {
	db *gorm.DB
}

func NewResetRepo(db *gorm.DB) IResetRepo {
	return &resetRepo{db: db}
}

func (r *resetRepo) Create(ctx context.Context, row *model.ResetToken) error {
	return dbFromCtx(ctx, r.db).Create(row).Error
}

func (r *resetRepo) GetValidHash(ctx context.Context, hash string, now time.Time) (model.ResetToken, error) {
	return getOne[model.ResetToken](dbFromCtx(ctx, r.db).
		Where("token_hash = ? AND used_at IS NULL AND expires_at > ?", hash, now))
}

func (r *resetRepo) MarkUsed(ctx context.Context, id uuid.UUID, at time.Time) error {
	return dbFromCtx(ctx, r.db).Model(&model.ResetToken{}).
		Where("id = ?", id).
		Update("used_at", at).Error
}
