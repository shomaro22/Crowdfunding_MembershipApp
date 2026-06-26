package repository

import (
	"context"

	"crowdfund-app/entity"

	"gorm.io/gorm"
)

type ActivityRepo interface {
	Create(ctx context.Context, row *entity.ActivityLog) error
}

type activityRepo struct {
	db *gorm.DB
}

func NewActivityRepo(db *gorm.DB) ActivityRepo {
	return &activityRepo{db: db}
}

func (r *activityRepo) Create(ctx context.Context, row *entity.ActivityLog) error {
	return dbFromCtx(ctx, r.db).Create(row).Error
}
