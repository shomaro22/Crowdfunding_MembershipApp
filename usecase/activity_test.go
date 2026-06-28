package usecase_test

import (
	"context"
	"testing"

	"crowdfund-app/entity"
	"crowdfund-app/repository"
	"crowdfund-app/usecase"

	"github.com/google/uuid"
)

type stubActivityRepo struct {
	last *entity.ActivityLog
}

func (s *stubActivityRepo) Create(_ context.Context, row *entity.ActivityLog) error {
	s.last = row
	return nil
}

func (s *stubActivityRepo) List(_ context.Context, _ repository.ActivityListFilter) ([]entity.ActivityLog, int64, error) {
	return nil, 0, nil
}

func TestLogRateLimitDenied(t *testing.T) {
	var (
		repo   stubActivityRepo
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
	if repo.last.Action != entity.SecRateLimit {
		t.Fatalf("action=%q want %q", repo.last.Action, entity.SecRateLimit)
	}
	if repo.last.UserID == nil || *repo.last.UserID != uid {
		t.Fatalf("user_id=%v want %v", repo.last.UserID, uid)
	}
	if repo.last.Resource == nil || *repo.last.Resource != "rate_limit" {
		t.Fatalf("resource=%v", repo.last.Resource)
	}
}
