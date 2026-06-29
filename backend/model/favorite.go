package model

import (
	"time"

	"github.com/google/uuid"
)

type Favorite struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_favorite_user_project,priority:1"`
	ProjectID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_favorite_user_project,priority:2"`
	CreatedAt time.Time `gorm:"not null"`
}

func (Favorite) TableName() string { return "favorites" }
