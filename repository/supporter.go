package repository

import (
	"context"
	"errors"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SupporterRepo interface {
	FindActive(ctx context.Context, projectID, userID uuid.UUID) (*entity.ProjectSupporter, error)
	FindMembership(ctx context.Context, projectID, userID uuid.UUID) (*entity.ProjectSupporter, error)
	FindStripeSubscriptionID(ctx context.Context, subscriptionID string) (*entity.ProjectSupporter, error)
	FindID(ctx context.Context, id uuid.UUID) (*entity.ProjectSupporter, error)
	Create(ctx context.Context, row *entity.ProjectSupporter) error
	Update(ctx context.Context, row *entity.ProjectSupporter) error
}

type supporterRepo struct {
	db *gorm.DB
}

func NewSupporterRepo(db *gorm.DB) SupporterRepo {
	return &supporterRepo{db: db}
}

func (r *supporterRepo) dbCtx(ctx context.Context) *gorm.DB {
	return dbFromCtx(ctx, r.db)
}

func (r *supporterRepo) FindActive(ctx context.Context, projectID, userID uuid.UUID) (*entity.ProjectSupporter, error) {
	var (
		row entity.ProjectSupporter
		err error
	)
	err = r.dbCtx(ctx).
		Where("project_id = ? AND user_id = ? AND status = ?", projectID, userID, entity.SupporterStatusActive).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *supporterRepo) FindMembership(ctx context.Context, projectID, userID uuid.UUID) (*entity.ProjectSupporter, error) {
	var (
		row entity.ProjectSupporter
		err error
	)
	err = r.dbCtx(ctx).
		Where("project_id = ? AND user_id = ? AND status IN ?", projectID, userID,
			[]entity.SupporterStatus{entity.SupporterStatusActive, entity.SupporterStatusPendingRenewal}).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *supporterRepo) FindStripeSubscriptionID(ctx context.Context, subscriptionID string) (*entity.ProjectSupporter, error) {
	var (
		row entity.ProjectSupporter
		err error
	)
	err = r.dbCtx(ctx).Where("stripe_subscription_id = ?", subscriptionID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *supporterRepo) FindID(ctx context.Context, id uuid.UUID) (*entity.ProjectSupporter, error) {
	var (
		row entity.ProjectSupporter
		err error
	)
	err = r.dbCtx(ctx).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *supporterRepo) Create(ctx context.Context, row *entity.ProjectSupporter) error {
	return r.dbCtx(ctx).Create(row).Error
}

func (r *supporterRepo) Update(ctx context.Context, row *entity.ProjectSupporter) error {
	return r.dbCtx(ctx).Save(row).Error
}
