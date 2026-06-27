package repository

import (
	"context"
	"errors"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WSTokenRepo interface {
	Create(ctx context.Context, row *entity.WSToken) error
	FindValidHash(ctx context.Context, hash string, now time.Time) (*entity.WSToken, error)
	MarkUsed(ctx context.Context, id uuid.UUID, at time.Time) error
}

type wsTokenRepo struct {
	db *gorm.DB
}

func NewWSTokenRepo(db *gorm.DB) WSTokenRepo {
	return &wsTokenRepo{db: db}
}

func (r *wsTokenRepo) Create(ctx context.Context, row *entity.WSToken) error {
	return dbFromCtx(ctx, r.db).Create(row).Error
}

func (r *wsTokenRepo) FindValidHash(ctx context.Context, hash string, now time.Time) (*entity.WSToken, error) {
	var (
		m   entity.WSToken
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

func (r *wsTokenRepo) MarkUsed(ctx context.Context, id uuid.UUID, at time.Time) error {
	return dbFromCtx(ctx, r.db).Model(&entity.WSToken{}).
		Where("id = ?", id).
		Update("used_at", at).Error
}
