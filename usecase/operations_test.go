package usecase

import (
	"context"
	"testing"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"
)

func TestListActivityLogsRejectsInvalidActionViaValidator(t *testing.T) {
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

func TestListActivityLogsRejectsInvalidIPViaValidator(t *testing.T) {
	var (
		ops OperationsUseCase
		q   ActivityLogsQuery
		err error
		ip  string
	)
	ip = "not-an-ip"
	q.IP = &ip
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
