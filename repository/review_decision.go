package repository

import (
	"context"

	"crowdfund-app/entity"

	"gorm.io/gorm"
)

type ReviewDecisionRepo interface {
	Create(ctx context.Context, row *entity.ReviewDecision) error
}

type reviewDecisionRepo struct {
	db *gorm.DB
}

func NewReviewDecisionRepo(db *gorm.DB) ReviewDecisionRepo {
	return &reviewDecisionRepo{db: db}
}

func (r *reviewDecisionRepo) Create(ctx context.Context, row *entity.ReviewDecision) error {
	return dbFromCtx(ctx, r.db).Create(row).Error
}
