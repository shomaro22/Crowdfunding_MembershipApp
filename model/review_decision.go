package model

import (
	"time"

	"github.com/google/uuid"
)

type ReviewDecision struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ReviewerID   uuid.UUID `gorm:"type:uuid;not null;index"`
	ResourceType string    `gorm:"column:resource_type;not null;index"`
	ResourceID   uuid.UUID `gorm:"type:uuid;not null;index"`
	Decision     string    `gorm:"not null"`
	Reason       *string   `gorm:"type:text"`
	CreatedAt    time.Time `gorm:"not null"`
}

func (ReviewDecision) TableName() string { return "review_decisions" }

const (
	ReviewBuyerRegistration  = "user_registration"
	ReviewSellerRegistration = "seller_registration"
	ReviewProject            = "project"
	ReviewProjectDelete      = "project_delete"
)

const (
	ReviewApproved = "approved"
	ReviewRejected = "rejected"
)
