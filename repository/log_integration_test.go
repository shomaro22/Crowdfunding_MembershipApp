//go:build integration

package repository_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"crowdfund-app/db"
	"crowdfund-app/entity"
	"crowdfund-app/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func openIntegrationDB(t *testing.T) *gorm.DB {
	var (
		gdb *gorm.DB
		err error
	)
	t.Helper()
	if strings.TrimSpace(os.Getenv("DATABASE_URL")) == "" {
		t.Skip("DATABASE_URL が未設定のためスキップ")
	}
	gdb, err = db.Open(os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	return gdb
}

func TestActivityRepoList(t *testing.T) {
	var (
		gdb      *gorm.DB
		repo     repository.ActivityRepo
		ctx      context.Context
		rowID    uuid.UUID
		userID   uuid.UUID
		now      time.Time
		resource string
		ip       string
		rows     []entity.ActivityLog
		total    int64
		err      error
	)
	gdb = openIntegrationDB(t)
	repo = repository.NewActivityRepo(gdb)
	ctx = context.Background()
	rowID = uuid.New()
	userID = uuid.New()
	now = time.Now().UTC()
	resource = "rate_limit"
	ip = "203.0.113.10"
	err = repo.Create(ctx, &entity.ActivityLog{
		ID:         rowID,
		UserID:     &userID,
		Action:     entity.SecRateLimit,
		Resource:   &resource,
		IPAddress:  &ip,
		CreatedAt:  now,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = gdb.Delete(&entity.ActivityLog{}, "id = ?", rowID).Error
	})

	rows, total, err = repo.List(ctx, repository.ActivityListFilter{
		UserID: &userID,
		Action: activityActionPtr(entity.SecRateLimit),
		IP:     &ip,
		Limit:  10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total < 1 {
		t.Fatalf("total=%d want >= 1", total)
	}
	if len(rows) < 1 {
		t.Fatal("expected at least one row")
	}
}

func TestAuditRepoList(t *testing.T) {
	var (
		gdb    *gorm.DB
		repo   repository.AuditRepo
		ctx    context.Context
		rowID  uuid.UUID
		admin  uuid.UUID
		action string
		rows   []entity.AuditLog
		total  int64
		err    error
	)
	gdb = openIntegrationDB(t)
	repo = repository.NewAuditRepo(gdb)
	ctx = context.Background()
	rowID = uuid.New()
	admin = uuid.New()
	action = "batch.manual_run"
	err = repo.Create(ctx, &entity.AuditLog{
		ID:         rowID,
		AdminID:    admin,
		Action:     action,
		Resource:   "batch",
		ResourceID: "purge_tokens",
		CreatedAt:  time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = gdb.Delete(&entity.AuditLog{}, "id = ?", rowID).Error
	})

	rows, total, err = repo.List(ctx, repository.AuditListFilter{
		AdminID: &admin,
		Action:  &action,
		Limit:   10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total < 1 || len(rows) < 1 {
		t.Fatalf("total=%d rows=%d", total, len(rows))
	}
}

func activityActionPtr(a entity.ActivityAction) *entity.ActivityAction {
	return &a
}
