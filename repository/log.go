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

type AuditRepo interface {
	Create(ctx context.Context, row *entity.AuditLog) error
}

type auditRepo struct {
	db *gorm.DB
}

func NewAuditRepo(db *gorm.DB) AuditRepo {
	return &auditRepo{db: db}
}

func (r *auditRepo) Create(ctx context.Context, row *entity.AuditLog) error {
	return dbFromCtx(ctx, r.db).Create(row).Error
}

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
