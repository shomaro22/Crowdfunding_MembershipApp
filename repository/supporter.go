package repository

import (
	"context"
	"errors"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SupporterRepo interface {
	FindActive(ctx context.Context, projectID, userID uuid.UUID) (*entity.ProjectSupporter, error)
	FindMembership(ctx context.Context, projectID, userID uuid.UUID) (*entity.ProjectSupporter, error)
	FindStripeSubscriptionID(ctx context.Context, subscriptionID string) (*entity.ProjectSupporter, error)
	FindID(ctx context.Context, id uuid.UUID) (*entity.ProjectSupporter, error)
	FindOwned(ctx context.Context, id, userID uuid.UUID) (*entity.ProjectSupporter, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]entity.ProjectSupporter, error)
	ListDuePlanChanges(ctx context.Context, asOf time.Time) ([]entity.ProjectSupporter, error)
	ListExpirable(ctx context.Context, asOf time.Time) ([]entity.ProjectSupporter, error)
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

func (r *supporterRepo) FindOwned(ctx context.Context, id, userID uuid.UUID) (*entity.ProjectSupporter, error) {
	var (
		row entity.ProjectSupporter
		err error
	)
	err = r.dbCtx(ctx).Where("id = ? AND user_id = ?", id, userID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *supporterRepo) ListByUserID(ctx context.Context, userID uuid.UUID) ([]entity.ProjectSupporter, error) {
	var (
		rows []entity.ProjectSupporter
		err  error
	)
	err = r.dbCtx(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&rows).Error
	return rows, err
}

func (r *supporterRepo) ListDuePlanChanges(ctx context.Context, asOf time.Time) ([]entity.ProjectSupporter, error) {
	var (
		rows []entity.ProjectSupporter
		err  error
	)
	err = r.dbCtx(ctx).
		Where("plan_change_date IS NOT NULL AND plan_change_date <= ?", asOf).
		Where("pending_plan_id IS NOT NULL OR pending_billing_mode IS NOT NULL").
		Find(&rows).Error
	return rows, err
}

func (r *supporterRepo) ListExpirable(ctx context.Context, asOf time.Time) ([]entity.ProjectSupporter, error) {
	var (
		rows []entity.ProjectSupporter
		err  error
	)
	err = r.dbCtx(ctx).
		Where("current_period_end < ?", asOf).
		Where("status IN ?", []entity.SupporterStatus{
			entity.SupporterStatusActive,
			entity.SupporterStatusPendingRenewal,
			entity.SupporterStatusCancelled,
		}).
		Find(&rows).Error
	return rows, err
}

func (r *supporterRepo) Create(ctx context.Context, row *entity.ProjectSupporter) error {
	return r.dbCtx(ctx).Create(row).Error
}

func (r *supporterRepo) Update(ctx context.Context, row *entity.ProjectSupporter) error {
	return r.dbCtx(ctx).Save(row).Error
}
