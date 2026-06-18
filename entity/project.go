package entity

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID              uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	SellerID        uuid.UUID     `gorm:"type:uuid;not null;index"`
	Title           string        `gorm:"not null"`
	Description     string        `gorm:"type:text;not null"`
	TargetAmount    int64         `gorm:"column:target_amount;not null"`
	CurrentAmount   int64         `gorm:"column:current_amount;not null;default:0"`
	SupporterCount  int           `gorm:"column:supporter_count;not null;default:0"`
	DurationType    DurationType  `gorm:"column:duration_type;not null"`
	StartDate       time.Time     `gorm:"column:start_date;not null"`
	EndDate         *time.Time    `gorm:"column:end_date"`
	Status          ProjectStatus `gorm:"not null;default:PENDING"`
	MainImageURL    *string       `gorm:"column:main_image_url;type:varchar"`
	DeleteRequested bool          `gorm:"column:delete_requested;not null;default:false"`
	CreatedAt       time.Time     `gorm:"not null"`
	UpdatedAt       time.Time     `gorm:"not null"`
}

func (Project) TableName() string { return "projects" }

func (p Project) IsPublic(now time.Time) bool {
	if p.Status != ProjectStatusActive {
		return false
	}
	if p.DurationType == DurationLimited && p.EndDate != nil && p.EndDate.Before(now) {
		return false
	}
	return true
}

type ProjectPlan struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ProjectID      uuid.UUID      `gorm:"type:uuid;not null;index"`
	MembershipType MembershipType `gorm:"column:membership_type;not null"`
	Price          int            `gorm:"not null"`
	MaxSupporters  *int           `gorm:"column:max_supporters"`
	Benefits       string         `gorm:"type:text;not null"`
	IsActive       bool           `gorm:"column:is_active;not null;default:true"`
	SupporterCount int            `gorm:"column:supporter_count;not null;default:0"`
	CreatedAt      time.Time      `gorm:"not null"`
	UpdatedAt      time.Time      `gorm:"not null"`
}

func (ProjectPlan) TableName() string { return "project_plans" }

type ProjectImage struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ProjectID uuid.UUID `gorm:"type:uuid;not null;index"`
	ImageURL  string    `gorm:"column:image_url;not null"`
	IsMain    bool      `gorm:"column:is_main;not null;default:false"`
	SortOrder int       `gorm:"column:sort_order;not null;default:0"`
	CreatedAt time.Time `gorm:"not null"`
}

func (ProjectImage) TableName() string { return "project_images" }

type ProjectContent struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ProjectID        uuid.UUID      `gorm:"type:uuid;not null;index"`
	Title            string         `gorm:"not null"`
	Body             *string        `gorm:"type:text"`
	StorageKey       *string        `gorm:"column:storage_key;type:varchar"`
	ExternalURL      *string        `gorm:"column:external_url;type:varchar"`
	MinimumPlan      MembershipType `gorm:"column:minimum_plan;not null"`
	SortOrder        int            `gorm:"column:sort_order;not null;default:0"`
	CreatedAt        time.Time      `gorm:"not null"`
	UpdatedAt        time.Time      `gorm:"not null"`
}

func (ProjectContent) TableName() string { return "project_contents" }
