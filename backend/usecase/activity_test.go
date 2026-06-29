package usecase_test

import (
	"context"
	"testing"
	"time"

	"crowdfund-app/model"
	"crowdfund-app/repository"
	"crowdfund-app/usecase"

	"github.com/google/uuid"
)

type stubIActivityRepo struct {
	last *model.ActivityLog
}

func (s *stubIActivityRepo) Create(_ context.Context, row *model.ActivityLog) error {
	s.last = row
	return nil
}

func (s *stubIActivityRepo) List(_ context.Context, _ repository.ActivityListFilter) ([]model.ActivityLog, int64, error) {
	return nil, 0, nil
}

func (s *stubIActivityRepo) CountByActionBetween(_ context.Context, _ model.ActivityAction, _, _ time.Time) (int64, error) {
	return 0, nil
}

func (s *stubIActivityRepo) TopIPsBetween(_ context.Context, _, _ time.Time, _ int) ([]repository.ActivityIPCount, error) {
	return nil, nil
}

func TestLogRateLimitDenied(t *testing.T) {
	var (
		repo   stubIActivityRepo
		uid    uuid.UUID
		ctx    context.Context
	)
	uid = uuid.New()
	ctx = context.Background()
	usecase.LogRateLimitDenied(ctx, &repo, &uid, "login", "login:1.2.3.4", usecase.RequestMeta{
		IPAddress: "1.2.3.4",
		UserAgent: "test",
		RequestID: "req-1",
	})
	if repo.last == nil {
		t.Fatal("expected activity log row")
	}
	if repo.last.Action != model.SecRateLimit {
		t.Fatalf("action=%q want %q", repo.last.Action, model.SecRateLimit)
	}
	if repo.last.UserID == nil || *repo.last.UserID != uid {
		t.Fatalf("user_id=%v want %v", repo.last.UserID, uid)
	}
	if repo.last.Resource == nil || *repo.last.Resource != "rate_limit" {
		t.Fatalf("resource=%v", repo.last.Resource)
	}
}
