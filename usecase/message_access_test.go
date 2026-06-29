package usecase_test

import (
	"testing"
	"time"

	"crowdfund-app/model"
)

func TestCanMessageActiveSupporter(t *testing.T) {
	var (
		s   model.ProjectSupporter
		err error
		now time.Time
	)
	now = time.Now().UTC()
	s = model.ProjectSupporter{
		Status:           model.SupporterStatusActive,
		CurrentPeriodEnd: now.Add(24 * time.Hour),
	}
	err = model.CanMessage(s, now)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCanMessageExpiredBenefits(t *testing.T) {
	var (
		s   model.ProjectSupporter
		err error
		now time.Time
	)
	now = time.Now().UTC()
	s = model.ProjectSupporter{
		Status:           model.SupporterStatusExpired,
		CurrentPeriodEnd: now.Add(-time.Hour),
	}
	err = model.CanMessage(s, now)
	if err == nil {
		t.Fatal("expected error for expired supporter")
	}
}
