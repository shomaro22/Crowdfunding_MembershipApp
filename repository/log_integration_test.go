//go:build integration

package repository_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"crowdfund-app/auth"
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

func seedIntegrationUser(t *testing.T, gdb *gorm.DB, role entity.UserRole, adminRole *entity.AdminRole) *entity.User {
	var (
		hash string
		now  time.Time
		u    *entity.User
		err  error
	)
	t.Helper()
	hash, err = auth.HashPassword("password123")
	if err != nil {
		t.Fatal(err)
	}
	now = time.Now().UTC()
	u = &entity.User{
		ID:            uuid.New(),
		Email:         fmt.Sprintf("%s-%s@test.local", role, uuid.New()),
		PasswordHash:  hash,
		Role:          role,
		AccountStatus: entity.AccountStatusActive,
		SellerStatus:  entity.SellerStatusNotApplicable,
		AdminRole:     adminRole,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err = gdb.Create(u).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = gdb.Delete(&entity.User{}, "id = ?", u.ID).Error
	})
	return u
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
	userID = seedIntegrationUser(t, gdb, entity.UserRoleBuyer, nil).ID
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
	admin = seedIntegrationUser(t, gdb, entity.UserRoleAdmin, adminRolePtr(entity.AdminRoleSuperAdmin)).ID
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

func adminRolePtr(r entity.AdminRole) *entity.AdminRole {
	return &r
}
