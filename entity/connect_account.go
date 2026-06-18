package entity

import (
	"time"

	"github.com/google/uuid"
)

type ConnectAccount struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID           uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	StripeAccountID  string    `gorm:"column:stripe_account_id;not null;uniqueIndex"`
	ChargesEnabled   bool      `gorm:"column:charges_enabled;not null;default:false"`
	PayoutsEnabled   bool      `gorm:"column:payouts_enabled;not null;default:false"`
	DetailsSubmitted bool      `gorm:"column:details_submitted;not null;default:false"`
	CreatedAt        time.Time `gorm:"not null"`
	UpdatedAt        time.Time `gorm:"not null"`
}

func (ConnectAccount) TableName() string { return "connect_accounts" }

func (c ConnectAccount) CanCharge() bool {
	return c.ChargesEnabled
}
