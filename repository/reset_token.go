package repository

import (
	"context"
	"errors"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ResetRepo interface {
	Create(ctx context.Context, row *entity.ResetToken) error
	FindValidHash(ctx context.Context, hash string, now time.Time) (*entity.ResetToken, error)
	MarkUsed(ctx context.Context, id uuid.UUID, at time.Time) error
}

type resetRepo struct {
	db *gorm.DB
}

func NewResetRepo(db *gorm.DB) ResetRepo {
	return &resetRepo{db: db}
}

func (r *resetRepo) Create(ctx context.Context, row *entity.ResetToken) error {
	return dbFromCtx(ctx, r.db).Create(row).Error
}

func (r *resetRepo) FindValidHash(ctx context.Context, hash string, now time.Time) (*entity.ResetToken, error) {
	var (
		m   entity.ResetToken
		err error
	)
	err = dbFromCtx(ctx, r.db).
		Where("token_hash = ? AND used_at IS NULL AND expires_at > ?", hash, now).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *resetRepo) MarkUsed(ctx context.Context, id uuid.UUID, at time.Time) error {
	return dbFromCtx(ctx, r.db).Model(&entity.ResetToken{}).
		Where("id = ?", id).
		Update("used_at", at).Error
}
