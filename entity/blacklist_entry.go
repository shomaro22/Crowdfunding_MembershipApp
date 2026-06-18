package entity

import (
	"time"

	"github.com/google/uuid"
)

type BlacklistEntry struct {
	ID                     uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	EmailHash              string     `gorm:"column:email_hash;not null;uniqueIndex"`
	StripeCustomerID       *string    `gorm:"column:stripe_customer_id;type:varchar"`
	StripeConnectAccountID *string    `gorm:"column:stripe_connect_account_id;type:varchar"`
	Reason                 *string    `gorm:"type:text"`
	BannedBy               *uuid.UUID `gorm:"type:uuid;column:banned_by"`
	CreatedAt              time.Time  `gorm:"not null"`
}

func (BlacklistEntry) TableName() string { return "blacklist_entries" }
