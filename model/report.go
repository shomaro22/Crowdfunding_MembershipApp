package model

import (
	"time"

	"github.com/google/uuid"
)

type Report struct {
	ID           uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ReporterID   uuid.UUID        `gorm:"type:uuid;not null;index"`
	TargetType   ReportTargetType `gorm:"column:target_type;not null"`
	TargetID     uuid.UUID        `gorm:"type:uuid;not null;index"`
	Reason       string           `gorm:"not null"`
	Detail       *string          `gorm:"type:text"`
	Status       ReportStatus     `gorm:"not null;default:PENDING"`
	ReviewedBy   *uuid.UUID       `gorm:"type:uuid;column:reviewed_by"`
	ReviewedAt   *time.Time       `gorm:"column:reviewed_at"`
	Resolution   *string          `gorm:"type:text"`
	CreatedAt    time.Time        `gorm:"not null;index"`
	UpdatedAt    time.Time        `gorm:"not null"`
}

func (Report) TableName() string { return "reports" }
