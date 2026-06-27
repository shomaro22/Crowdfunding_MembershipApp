package entity

import (
	"time"

	"github.com/google/uuid"
)

type BatchRun struct {
	ID           uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	JobName      string      `gorm:"column:job_name;not null;index"`
	Status       BatchStatus `gorm:"not null"`
	StartedAt    time.Time   `gorm:"column:started_at;not null;index"`
	FinishedAt   *time.Time  `gorm:"column:finished_at"`
	Processed    int         `gorm:"not null;default:0"`
	Failed       int         `gorm:"not null;default:0"`
	ErrorMessage *string     `gorm:"column:error_message;type:text"`
	TriggeredBy  string      `gorm:"column:triggered_by;not null"`
	DryRun       bool        `gorm:"column:dry_run;not null;default:false"`
}

func (BatchRun) TableName() string { return "batch_job_runs" }

const (
	ApplyPlans       = "apply_plans"
	ExpireSupporters = "expire_supporters"
	ReconcileStripe  = "reconcile_stripe"
)
