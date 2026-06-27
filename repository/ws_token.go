package repository

import (
	"context"
	"errors"
	"time"

	"crowdfund-app/entity"

	"gorm.io/gorm"
)

type WSTokenRepo interface {
	Create(ctx context.Context, row *entity.WSToken) error
	ConsumeValidHash(ctx context.Context, hash string, now time.Time) (*entity.WSToken, error)
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

func (r *wsTokenRepo) ConsumeValidHash(ctx context.Context, hash string, now time.Time) (*entity.WSToken, error) {
	var (
		tx  *gorm.DB
		row entity.WSToken
		res *gorm.DB
		err error
	)
	tx = dbFromCtx(ctx, r.db).Begin()
	if err = tx.Error; err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	err = tx.
		Where("token_hash = ? AND used_at IS NULL AND expires_at > ?", hash, now).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = gorm.ErrRecordNotFound
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	res = tx.Model(&entity.WSToken{}).
		Where("id = ? AND used_at IS NULL", row.ID).
		Update("used_at", now)
	if res.Error != nil {
		err = res.Error
		return nil, err
	}
	if res.RowsAffected != 1 {
		err = gorm.ErrRecordNotFound
		return nil, nil
	}
	if err = tx.Commit().Error; err != nil {
		return nil, err
	}
	return &row, nil
}
