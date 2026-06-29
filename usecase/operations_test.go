package usecase

import (
	"context"
	"testing"
	"time"

	"crowdfund-app/model"
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
	ops = OperationsUseCase{Activity: stubIActivityRepoForOps{}}
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
	ops = OperationsUseCase{Activity: stubIActivityRepoForOps{}}
	_, err = ops.ListActivityLogs(t.Context(), q)
	if err != ErrBadRequest {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}
}

type stubIActivityRepoForOps struct{}

func (stubIActivityRepoForOps) Create(_ context.Context, _ *model.ActivityLog) error {
	return nil
}

func (stubIActivityRepoForOps) List(_ context.Context, _ repository.ActivityListFilter) ([]model.ActivityLog, int64, error) {
	return nil, 0, nil
}

func (stubIActivityRepoForOps) CountByActionBetween(_ context.Context, _ model.ActivityAction, _, _ time.Time) (int64, error) {
	return 0, nil
}

func (stubIActivityRepoForOps) TopIPsBetween(_ context.Context, _, _ time.Time, _ int) ([]repository.ActivityIPCount, error) {
	return nil, nil
}
