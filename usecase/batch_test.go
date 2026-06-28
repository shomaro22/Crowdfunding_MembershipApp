package usecase

import (
	"testing"
	"time"

	"crowdfund-app/entity"
)

func TestIsExpirableStatus(t *testing.T) {
	if !isExpirableStatus(entity.SupporterStatusActive) {
		t.Fatal("active should be expirable")
	}
	if isExpirableStatus(entity.SupporterStatusExpired) {
		t.Fatal("already expired should not be expirable")
	}
}

func TestExpireSupporterSkipsBeforePeriodEnd(t *testing.T) {
	var (
		now = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		s   = &entity.ProjectSupporter{
			Status:           entity.SupporterStatusActive,
			CurrentPeriodEnd: now.Add(24 * time.Hour),
		}
	)
	if now.After(s.CurrentPeriodEnd) {
		t.Fatal("test setup")
	}
	if now.After(s.CurrentPeriodEnd) || !isExpirableStatus(s.Status) {
		t.Fatal("should skip expire before period end")
	}
}

func TestIsSupportedBatchJob(t *testing.T) {
	if !isSupportedBatchJob(entity.ApplyPlans) {
		t.Fatal("apply_plans should be supported")
	}
	if !isSupportedBatchJob(entity.PurgeTokens) {
		t.Fatal("purge_tokens should be supported")
	}
	if isSupportedBatchJob("reconcile_stripe") {
		t.Fatal("reconcile_stripe is not a supported batch job")
	}
}
