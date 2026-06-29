package repository

import (
	"context"
	"time"

	"crowdfund-app/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SupporterRepo interface {
	FindActive(ctx context.Context, projectID, userID uuid.UUID) (*model.ProjectSupporter, error)
	FindMembership(ctx context.Context, projectID, userID uuid.UUID) (*model.ProjectSupporter, error)
	FindStripeSubscriptionID(ctx context.Context, subscriptionID string) (*model.ProjectSupporter, error)
	FindID(ctx context.Context, id uuid.UUID) (*model.ProjectSupporter, error)
	FindOwned(ctx context.Context, id, userID uuid.UUID) (*model.ProjectSupporter, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.ProjectSupporter, error)
	ListDuePlanChanges(ctx context.Context, asOf time.Time) ([]model.ProjectSupporter, error)
	ListExpirable(ctx context.Context, asOf time.Time) ([]model.ProjectSupporter, error)
	Create(ctx context.Context, row *model.ProjectSupporter) error
	Update(ctx context.Context, row *model.ProjectSupporter) error
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

func (r *supporterRepo) FindActive(ctx context.Context, projectID, userID uuid.UUID) (*model.ProjectSupporter, error) {
	return firstOnePtr[model.ProjectSupporter](r.dbCtx(ctx).
		Where("project_id = ? AND user_id = ? AND status = ?", projectID, userID, model.SupporterStatusActive))
}

func (r *supporterRepo) FindMembership(ctx context.Context, projectID, userID uuid.UUID) (*model.ProjectSupporter, error) {
	return firstOnePtr[model.ProjectSupporter](r.dbCtx(ctx).
		Where("project_id = ? AND user_id = ? AND status IN ?", projectID, userID,
			[]model.SupporterStatus{model.SupporterStatusActive, model.SupporterStatusPendingRenewal}))
}

func (r *supporterRepo) FindStripeSubscriptionID(ctx context.Context, subscriptionID string) (*model.ProjectSupporter, error) {
	return firstOnePtr[model.ProjectSupporter](r.dbCtx(ctx).Where("stripe_subscription_id = ?", subscriptionID))
}

func (r *supporterRepo) FindID(ctx context.Context, id uuid.UUID) (*model.ProjectSupporter, error) {
	return firstOnePtr[model.ProjectSupporter](r.dbCtx(ctx).Where("id = ?", id))
}

func (r *supporterRepo) FindOwned(ctx context.Context, id, userID uuid.UUID) (*model.ProjectSupporter, error) {
	return firstOnePtr[model.ProjectSupporter](r.dbCtx(ctx).Where("id = ? AND user_id = ?", id, userID))
}

func (r *supporterRepo) ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.ProjectSupporter, error) {
	var (
		rows []model.ProjectSupporter
		err  error
	)
	err = r.dbCtx(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&rows).Error
	return rows, err
}

func (r *supporterRepo) ListDuePlanChanges(ctx context.Context, asOf time.Time) ([]model.ProjectSupporter, error) {
	var (
		rows []model.ProjectSupporter
		err  error
	)
	err = r.dbCtx(ctx).
		Where("plan_change_date IS NOT NULL AND plan_change_date <= ?", asOf).
		Where("pending_plan_id IS NOT NULL OR pending_billing_mode IS NOT NULL").
		Find(&rows).Error
	return rows, err
}

func (r *supporterRepo) ListExpirable(ctx context.Context, asOf time.Time) ([]model.ProjectSupporter, error) {
	var (
		rows []model.ProjectSupporter
		err  error
	)
	err = r.dbCtx(ctx).
		Where("current_period_end < ?", asOf).
		Where("status IN ?", []model.SupporterStatus{
			model.SupporterStatusActive,
			model.SupporterStatusPendingRenewal,
			model.SupporterStatusCancelled,
		}).
		Find(&rows).Error
	return rows, err
}

func (r *supporterRepo) Create(ctx context.Context, row *model.ProjectSupporter) error {
	return r.dbCtx(ctx).Create(row).Error
}

func (r *supporterRepo) Update(ctx context.Context, row *model.ProjectSupporter) error {
	return r.dbCtx(ctx).Save(row).Error
}