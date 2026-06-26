package usecase

import (
	"testing"
	"time"

	"crowdfund-app/entity"
)

func TestSupporterCanManage(t *testing.T) {
	var (
		now = time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
		end = time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
		s   = &entity.ProjectSupporter{
			Status:           entity.SupporterStatusActive,
			CurrentPeriodEnd: end,
		}
	)
	if !supporterCanManage(s, now) {
		t.Fatal("active supporter within period should be manageable")
	}
	s.Status = entity.SupporterStatusCancelled
	if !supporterCanManage(s, now) {
		t.Fatal("cancelled supporter within period should be manageable")
	}
	if supporterCanManage(s, end.Add(24*time.Hour)) {
		t.Fatal("expired cancelled supporter should not be manageable")
	}
	s.Status = entity.SupporterStatusExpired
	if supporterCanManage(s, now) {
		t.Fatal("expired status should not be manageable")
	}
}
