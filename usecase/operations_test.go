package usecase

import (
	"context"
	"testing"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"
)

func TestClampTimeRangeRejectsInverted(t *testing.T) {
	var (
		start time.Time
		end   time.Time
		err   error
	)
	start = time.Now().UTC()
	end = start.Add(-time.Hour)
	_, _, err = clampTimeRange(start, end)
	if err != ErrBadRequest {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}
}

func TestClampTimeRangeRejectsTooWide(t *testing.T) {
	var (
		start time.Time
		end   time.Time
		err   error
	)
	start = time.Now().UTC().Add(-100 * 24 * time.Hour)
	end = time.Now().UTC()
	_, _, err = clampTimeRange(start, end)
	if err != ErrBadRequest {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}
}

func TestListActivityLogsRejectsInvalidAction(t *testing.T) {
	var (
		ops OperationsUseCase
		q   ActivityLogsQuery
		err error
		bad string
	)
	bad = "not.a.real.action"
	q.Action = &bad
	ops = OperationsUseCase{Activity: stubActivityRepoForOps{}}
	_, err = ops.ListActivityLogs(t.Context(), q)
	if err != ErrBadRequest {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}
}

type stubActivityRepoForOps struct{}

func (stubActivityRepoForOps) Create(_ context.Context, _ *entity.ActivityLog) error {
	return nil
}

func (stubActivityRepoForOps) List(_ context.Context, _ repository.ActivityListFilter) ([]entity.ActivityLog, int64, error) {
	return nil, 0, nil
}

func (stubActivityRepoForOps) CountByActionBetween(_ context.Context, _ entity.ActivityAction, _, _ time.Time) (int64, error) {
	return 0, nil
}

func (stubActivityRepoForOps) TopIPsBetween(_ context.Context, _, _ time.Time, _ int) ([]repository.ActivityIPCount, error) {
	return nil, nil
}
