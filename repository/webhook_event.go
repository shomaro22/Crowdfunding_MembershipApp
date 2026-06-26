package repository

import (
	"context"
	"errors"

	"crowdfund-app/entity"

	"gorm.io/gorm"
)

type WebhookEventRepo interface {
	Exists(ctx context.Context, stripeEventID string) (bool, error)
	Create(ctx context.Context, row *entity.WebhookEvent) error
}

type webhookEventRepo struct {
	db *gorm.DB
}

func NewWebhookEventRepo(db *gorm.DB) WebhookEventRepo {
	return &webhookEventRepo{db: db}
}

func (r *webhookEventRepo) dbCtx(ctx context.Context) *gorm.DB {
	return dbFromCtx(ctx, r.db)
}

func (r *webhookEventRepo) Exists(ctx context.Context, stripeEventID string) (bool, error) {
	var (
		err   error
		count int64
	)
	err = r.dbCtx(ctx).Model(&entity.WebhookEvent{}).
		Where("stripe_event_id = ?", stripeEventID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *webhookEventRepo) Create(ctx context.Context, row *entity.WebhookEvent) error {
	err := r.dbCtx(ctx).Create(row).Error
	if err != nil && errors.Is(err, gorm.ErrDuplicatedKey) {
		return nil
	}
	return err
}
