package entity

import (
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AdminID    uuid.UUID `gorm:"type:uuid;column:admin_id;not null;index"`
	Action     string    `gorm:"type:varchar(128);not null"`
	Resource   string    `gorm:"type:varchar(64);not null"`
	ResourceID string    `gorm:"column:resource_id;type:varchar(128);not null"`
	Details    *string   `gorm:"type:text"`
	Reason     *string   `gorm:"type:text"`
	CreatedAt  time.Time `gorm:"not null;index"`
}

func (AuditLog) TableName() string { return "audit_logs" }
