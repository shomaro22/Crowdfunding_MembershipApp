package repository

import (
	"context"
	"errors"

	"crowdfund-app/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FavoriteRepo interface {
	ListByUser(ctx context.Context, userID uuid.UUID) ([]model.Favorite, error)
	Exists(ctx context.Context, userID, projectID uuid.UUID) (bool, error)
	Create(ctx context.Context, row *model.Favorite) error
	Delete(ctx context.Context, userID, projectID uuid.UUID) (bool, error)
}

type favoriteRepo struct {
	db *gorm.DB
}

func NewFavoriteRepo(db *gorm.DB) FavoriteRepo {
	return &favoriteRepo{db: db}
}

func (r *favoriteRepo) dbCtx(ctx context.Context) *gorm.DB {
	return dbFromCtx(ctx, r.db)
}

func (r *favoriteRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.Favorite, error) {
	var (
		rows []model.Favorite
		err  error
	)
	err = r.dbCtx(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *favoriteRepo) Exists(ctx context.Context, userID, projectID uuid.UUID) (bool, error) {
	var (
		n   int64
		err error
	)
	err = r.dbCtx(ctx).Model(&model.Favorite{}).
		Where("user_id = ? AND project_id = ?", userID, projectID).
		Count(&n).Error
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *favoriteRepo) Create(ctx context.Context, row *model.Favorite) error {
	return r.dbCtx(ctx).Create(row).Error
}

func (r *favoriteRepo) Delete(ctx context.Context, userID, projectID uuid.UUID) (bool, error) {
	var (
		res *gorm.DB
	)
	res = r.dbCtx(ctx).
		Where("user_id = ? AND project_id = ?", userID, projectID).
		Delete(&model.Favorite{})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func IsFavoriteDuplicate(err error) bool {
	return errors.Is(err, gorm.ErrDuplicatedKey)
}
