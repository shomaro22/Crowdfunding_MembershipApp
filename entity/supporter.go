package entity

import (
	"time"

	"github.com/google/uuid"
)

// OpenAPI の end_date は DB の current_period_end に対応する。
type ProjectSupporter struct {
	ID                 uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ProjectID          uuid.UUID       `gorm:"type:uuid;not null;index:idx_supporter_project_user,priority:1"`
	UserID             uuid.UUID       `gorm:"type:uuid;not null;index:idx_supporter_project_user,priority:2"`
	PlanID             uuid.UUID       `gorm:"type:uuid;not null"`
	Status             SupporterStatus `gorm:"not null;default:ACTIVE"`
	BillingMode        BillingMode     `gorm:"column:billing_mode;not null"`
	CurrentPeriodStart time.Time       `gorm:"column:current_period_start;not null"`
	CurrentPeriodEnd   time.Time       `gorm:"column:current_period_end;not null"`
	PendingPlanID        *uuid.UUID      `gorm:"type:uuid;column:pending_plan_id"`
	PlanChangeDate       *time.Time      `gorm:"column:plan_change_date"`
	PendingBillingMode   *BillingMode    `gorm:"column:pending_billing_mode;type:varchar"`
	StripeCustomerID     *string         `gorm:"column:stripe_customer_id;type:varchar"`
	StripeSubscriptionID *string       `gorm:"column:stripe_subscription_id;type:varchar"`
	CreatedAt          time.Time       `gorm:"not null"`
	UpdatedAt          time.Time       `gorm:"not null"`
}

func (ProjectSupporter) TableName() string { return "project_supporters" }

// PENDING_RENEWAL でも current_period_end までは true。
func (s ProjectSupporter) HasBenefits(now time.Time) bool {
	switch s.Status {
	case SupporterStatusActive, SupporterStatusPendingRenewal:
		return !now.After(s.CurrentPeriodEnd)
	case SupporterStatusCancelled:
		return !now.After(s.CurrentPeriodEnd)
	default:
		return false
	}
}

func (s ProjectSupporter) IsActive(now time.Time) bool {
	return s.Status == SupporterStatusActive && !now.After(s.CurrentPeriodEnd)
}
