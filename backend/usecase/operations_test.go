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
		ops    OperationsUseCase
		repo   recordingActivityRepoForOps
		q      ActivityLogsQuery
		err    error
		bad    string
	)
	bad = "not.a.real.action"
	q.Action = &bad
	ops = OperationsUseCase{Activity: &repo}
	_, err = ops.ListActivityLogs(t.Context(), q)
	if err != ErrBadRequest {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}
	if repo.listCalls != 0 {
		t.Fatalf("List should not be called on validation error, calls=%d", repo.listCalls)
	}
}

func TestListActivityLogsRejectsInvalidIPViaValidator(t *testing.T) {
	var (
		ops  OperationsUseCase
		repo recordingActivityRepoForOps
		q    ActivityLogsQuery
		err  error
		ip   string
	)
	ip = "not-an-ip"
	q.IP = &ip
	ops = OperationsUseCase{Activity: &repo}
	_, err = ops.ListActivityLogs(t.Context(), q)
	if err != ErrBadRequest {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}
	if repo.listCalls != 0 {
		t.Fatalf("List should not be called on validation error, calls=%d", repo.listCalls)
	}
}

type recordingActivityRepoForOps struct {
	listCalls int
}

func (r *recordingActivityRepoForOps) Create(_ context.Context, _ *model.ActivityLog) error {
	return nil
}

func (r *recordingActivityRepoForOps) List(_ context.Context, _ repository.ActivityListFilter) ([]model.ActivityLog, int64, error) {
	r.listCalls++
	return nil, 0, nil
}

func (r *recordingActivityRepoForOps) CountByActionBetween(_ context.Context, _ model.ActivityAction, _, _ time.Time) (int64, error) {
	return 0, nil
}

func (r *recordingActivityRepoForOps) TopIPsBetween(_ context.Context, _, _ time.Time, _ int) ([]repository.ActivityIPCount, error) {
	return nil, nil
}
