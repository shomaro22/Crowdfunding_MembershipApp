//go:build integration

package repository_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"crowdfund-app/auth"
	"crowdfund-app/db"
	"crowdfund-app/model"
	"crowdfund-app/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	var (
		code int
		err  error
	)
	if strings.TrimSpace(os.Getenv("DATABASE_URL")) != "" {
		err = db.Up(os.Getenv("DATABASE_URL"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "integration migrations: %v\n", err)
			os.Exit(1)
		}
	}
	code = m.Run()
	os.Exit(code)
}

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
	t.Cleanup(func() {
		var sqlDB *sql.DB
		sqlDB, err = gdb.DB()
		if err == nil && sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	return gdb
}

func seedIntegrationUser(t *testing.T, gdb *gorm.DB, role model.UserRole, adminRole *model.AdminRole) *model.User {
	var (
		hash string
		now  time.Time
		u    *model.User
		err  error
	)
	t.Helper()
	hash, err = auth.HashPassword("password123")
	if err != nil {
		t.Fatal(err)
	}
	now = time.Now().UTC()
	u = &model.User{
		ID:            uuid.New(),
		Email:         fmt.Sprintf("%s-%s@test.local", role, uuid.New()),
		PasswordHash:  hash,
		Role:          role,
		AccountStatus: model.AccountStatusActive,
		SellerStatus:  model.SellerStatusNotApplicable,
		AdminRole:     adminRole,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err = gdb.Create(u).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = gdb.Delete(&model.User{}, "id = ?", u.ID).Error
	})
	return u
}

func TestIActivityRepoList(t *testing.T) {
	var (
		gdb      *gorm.DB
		repo     repository.IActivityRepo
		ctx      context.Context
		rowID    uuid.UUID
		userID   uuid.UUID
		now      time.Time
		resource string
		ip       string
		rows     []model.ActivityLog
		total    int64
		err      error
	)
	gdb = openIntegrationDB(t)
	repo = repository.NewActivityRepo(gdb)
	ctx = context.Background()
	rowID = uuid.New()
	userID = seedIntegrationUser(t, gdb, model.UserRoleBuyer, nil).ID
	now = time.Now().UTC()
	resource = "rate_limit"
	ip = "203.0.113.10"
	err = repo.Create(ctx, &model.ActivityLog{
		ID:         rowID,
		UserID:     &userID,
		Action:     model.SecRateLimit,
		Resource:   &resource,
		IPAddress:  &ip,
		CreatedAt:  now,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = gdb.Delete(&model.ActivityLog{}, "id = ?", rowID).Error
	})

	rows, total, err = repo.List(ctx, repository.ActivityListFilter{
		UserID: &userID,
		Action: activityActionPtr(model.SecRateLimit),
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

func TestIAuditRepoList(t *testing.T) {
	var (
		gdb    *gorm.DB
		repo   repository.IAuditRepo
		ctx    context.Context
		rowID  uuid.UUID
		admin  uuid.UUID
		action string
		rows   []model.AuditLog
		total  int64
		err    error
	)
	gdb = openIntegrationDB(t)
	repo = repository.NewAuditRepo(gdb)
	ctx = context.Background()
	rowID = uuid.New()
	admin = seedIntegrationUser(t, gdb, model.UserRoleAdmin, adminRolePtr(model.AdminRoleSuperAdmin)).ID
	action = "batch.manual_run"
	err = repo.Create(ctx, &model.AuditLog{
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
		_ = gdb.Delete(&model.AuditLog{}, "id = ?", rowID).Error
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

func activityActionPtr(a model.ActivityAction) *model.ActivityAction {
	return &a
}

func adminRolePtr(r model.AdminRole) *model.AdminRole {
	return &r
}
