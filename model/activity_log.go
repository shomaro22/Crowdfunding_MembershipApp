package model

import (
	"time"

	"github.com/google/uuid"
)

type ActivityLog struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID     *uuid.UUID     `gorm:"type:uuid;index:idx_activity_logs_user_created,priority:1"`
	Action     ActivityAction `gorm:"type:varchar(64);not null;index:idx_activity_logs_action_created,priority:1"`
	Resource   *string        `gorm:"type:varchar(64)"`
	ResourceID *string        `gorm:"column:resource_id;type:varchar(128)"`
	IPAddress  *string        `gorm:"column:ip_address;type:varchar"`
	UserAgent  *string        `gorm:"column:user_agent;type:text"`
	RequestID  *string        `gorm:"column:request_id;type:varchar(64)"`
	Metadata   JSONMetadata   `gorm:"type:jsonb"`
	CreatedAt  time.Time      `gorm:"not null;index:idx_activity_logs_user_created,priority:2;index:idx_activity_logs_action_created,priority:2"`
}

func (ActivityLog) TableName() string { return "activity_logs" }
