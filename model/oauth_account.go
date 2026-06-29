package model

import (
	"time"

	"github.com/google/uuid"
)

type OAuthAccount struct {
	ID          uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID      uuid.UUID     `gorm:"type:uuid;not null;index"`
	Provider    OAuthProvider `gorm:"not null"`
	ProviderSub string        `gorm:"column:provider_sub;not null"`
	Email       string        `gorm:"not null"`
	CreatedAt   time.Time     `gorm:"not null"`
}

func (OAuthAccount) TableName() string { return "oauth_accounts" }
