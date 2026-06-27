package usecase_test

import (
	"testing"
	"time"

	"crowdfund-app/entity"
)

func TestCanMessageActiveSupporter(t *testing.T) {
	var (
		s   entity.ProjectSupporter
		err error
		now time.Time
	)
	now = time.Now().UTC()
	s = entity.ProjectSupporter{
		Status:           entity.SupporterStatusActive,
		CurrentPeriodEnd: now.Add(24 * time.Hour),
	}
	err = entity.CanMessage(s, now)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCanMessageExpiredBenefits(t *testing.T) {
	var (
		s   entity.ProjectSupporter
		err error
		now time.Time
	)
	now = time.Now().UTC()
	s = entity.ProjectSupporter{
		Status:           entity.SupporterStatusExpired,
		CurrentPeriodEnd: now.Add(-time.Hour),
	}
	err = entity.CanMessage(s, now)
	if err == nil {
		t.Fatal("expected error for expired supporter")
	}
}
