package entity

import (
	"time"

	"github.com/google/uuid"
)

// Purchase は1決済の不変レコード。金額・Stripe ID は更新しない。
type Purchase struct {
	ID                      uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ProjectID               uuid.UUID      `gorm:"type:uuid;not null;index"`
	SupporterID             *uuid.UUID     `gorm:"type:uuid;index"`
	BuyerID                 uuid.UUID      `gorm:"type:uuid;not null;index"`
	Quantity                int            `gorm:"not null;default:1"`
	TotalPrice              int            `gorm:"column:total_price;not null"`
	Status                  PurchaseStatus `gorm:"not null;default:PENDING"`
	PaymentMethod           PaymentMethod  `gorm:"column:payment_method;not null"`
	PeriodStart             *time.Time     `gorm:"column:period_start"`
	PeriodEnd               *time.Time     `gorm:"column:period_end"`
	StripeInvoiceID       *string        `gorm:"column:stripe_invoice_id;type:varchar"`
	StripeIntentID        *string        `gorm:"column:stripe_payment_intent_id;type:varchar;uniqueIndex"`
	StripeCheckoutID      *string        `gorm:"column:stripe_checkout_session_id;type:varchar;uniqueIndex"`
	IdempotencyKey          *string        `gorm:"column:idempotency_key;type:varchar;uniqueIndex"`
	CreatedAt               time.Time      `gorm:"not null"`
	UpdatedAt               time.Time      `gorm:"not null"`
}

func (Purchase) TableName() string { return "purchases" }

type PurchaseEvent struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PurchaseID uuid.UUID      `gorm:"type:uuid;not null;index"`
	FromStatus *PurchaseStatus `gorm:"column:from_status;type:varchar"`
	ToStatus   PurchaseStatus `gorm:"column:to_status;not null"`
	Reason     *string        `gorm:"type:text"`
	CreatedAt  time.Time      `gorm:"not null"`
}

func (PurchaseEvent) TableName() string { return "purchase_events" }
