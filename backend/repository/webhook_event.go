package repository

import (
	"context"
	"time"

	"crowdfund-app/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IWebhookEventRepo interface {
	Claim(ctx context.Context, stripeEventID, eventType string) (bool, error)
	Release(ctx context.Context, stripeEventID string) error
}

type webhookEventRepo struct {
	db *gorm.DB
}

func NewWebhookEventRepo(db *gorm.DB) IWebhookEventRepo {
	return &webhookEventRepo{db: db}
}

func (r *webhookEventRepo) dbCtx(ctx context.Context) *gorm.DB {
	return dbFromCtx(ctx, r.db)
}

func (r *webhookEventRepo) Claim(ctx context.Context, stripeEventID, eventType string) (bool, error) {
	var (
		row model.WebhookEvent
		res *gorm.DB
	)
	row = model.WebhookEvent{
		ID:            uuid.New(),
		StripeEventID: stripeEventID,
		EventType:     eventType,
		ProcessedAt:   time.Now().UTC(),
	}
	res = r.dbCtx(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "stripe_event_id"}},
		DoNothing: true,
	}).Create(&row)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *webhookEventRepo) Release(ctx context.Context, stripeEventID string) error {
	return r.dbCtx(ctx).Where("stripe_event_id = ?", stripeEventID).Delete(&model.WebhookEvent{}).Error
}
