package entity

import (
	"time"

	"github.com/google/uuid"
)

type WebhookEvent struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	StripeEventID string    `gorm:"column:stripe_event_id;not null;uniqueIndex"`
	EventType     string    `gorm:"column:event_type;not null;index"`
	ProcessedAt   time.Time `gorm:"column:processed_at;not null"`
	PayloadHash   *string   `gorm:"column:payload_hash;type:varchar(64)"`
}

func (WebhookEvent) TableName() string { return "stripe_webhook_events" }
