package model

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ProjectID  uuid.UUID `gorm:"type:uuid;not null;index"`
	SenderID   uuid.UUID `gorm:"type:uuid;not null;index"`
	ReceiverID uuid.UUID `gorm:"type:uuid;not null;index"`
	Content    string    `gorm:"type:text;not null"`
	IsRead     bool      `gorm:"column:is_read;not null;default:false"`
	ReadAt     *time.Time `gorm:"column:read_at"`
	CreatedAt  time.Time `gorm:"not null;index"`
}

func (Message) TableName() string { return "messages" }

const MaxMessageLength = 4000
