//go:build integration

package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func TestProjectBuyerCannotCreate(t *testing.T) {
	var (
		e   *echo.Echo
		gdb *gorm.DB
		u   *entity.User
		tok string
		rec *httptest.ResponseRecorder
	)
	e = setupSecurityApp(t)
	gdb = openSecurityDB(t)
	u = seedSecurityUser(t, gdb, seedOpts{Role: entity.UserRoleBuyer})
	tok, _ = issueSecurityToken(u, true)
	rec = securityPOST(e, "/projects", tok, `{"title":"t","description":"d"}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rec.Code)
	}
}

func TestProjectReviewAdminCanListProjects(t *testing.T) {
	var (
		e     *echo.Echo
		gdb   *gorm.DB
		u     *entity.User
		tok   string
		role  entity.AdminRole
		rec   *httptest.ResponseRecorder
	)
	e = setupSecurityApp(t)
	gdb = openSecurityDB(t)
	role = entity.AdminRoleReviewAdmin
	u = seedSecurityUser(t, gdb, seedOpts{
		Role:        entity.UserRoleAdmin,
		AdminRole:   &role,
		TOTPEnabled: true,
	})
	tok, _ = issueSecurityToken(u, true)
	rec = securityGET(e, "/admin/projects", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want 200", rec.Code)
	}
}

func TestProjectFinanceAdminCannotListProjects(t *testing.T) {
	var (
		e     *echo.Echo
		gdb   *gorm.DB
		u     *entity.User
		tok   string
		role  entity.AdminRole
		rec   *httptest.ResponseRecorder
	)
	e = setupSecurityApp(t)
	gdb = openSecurityDB(t)
	role = entity.AdminRoleFinanceAdmin
	u = seedSecurityUser(t, gdb, seedOpts{
		Role:        entity.UserRoleAdmin,
		AdminRole:   &role,
		TOTPEnabled: true,
	})
	tok, _ = issueSecurityToken(u, true)
	rec = securityGET(e, "/admin/projects", tok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rec.Code)
	}
}

func TestProjectPendingHiddenFromPublic(t *testing.T) {
	var (
		e       *echo.Echo
		gdb     *gorm.DB
		seller  *entity.User
		project    *entity.Project
		rec     *httptest.ResponseRecorder
	)
	e = setupSecurityApp(t)
	gdb = openSecurityDB(t)
	seller = seedSecurityUser(t, gdb, seedOpts{Role: entity.UserRoleSeller})
	project = seedSecurityProject(t, gdb, seller.ID, entity.ProjectStatusPending)
	rec = securityGET(e, "/projects/"+project.ID.String(), "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", rec.Code)
	}
}

func TestProjectSellerCanViewOwnPending(t *testing.T) {
	var (
		e      *echo.Echo
		gdb    *gorm.DB
		seller *entity.User
		project   *entity.Project
		tok    string
		rec    *httptest.ResponseRecorder
	)
	e = setupSecurityApp(t)
	gdb = openSecurityDB(t)
	seller = seedSecurityUser(t, gdb, seedOpts{
		Role:        entity.UserRoleSeller,
		TOTPEnabled: true,
	})
	project = seedSecurityProject(t, gdb, seller.ID, entity.ProjectStatusPending)
	tok, _ = issueSecurityToken(seller, true)
	rec = securityGET(e, "/projects/"+project.ID.String(), tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want 200", rec.Code)
	}
}

func TestProjectRejectBuyerRequiresReason(t *testing.T) {
	var (
		e       *echo.Echo
		gdb     *gorm.DB
		buyer   *entity.User
		admin   *entity.User
		tok     string
		role    entity.AdminRole
		rec     *httptest.ResponseRecorder
	)
	e = setupSecurityApp(t)
	gdb = openSecurityDB(t)
	buyer = seedSecurityUser(t, gdb, seedOpts{
		Role:          entity.UserRoleBuyer,
		AccountStatus: entity.AccountStatusPending,
	})
	role = entity.AdminRoleReviewAdmin
	admin = seedSecurityUser(t, gdb, seedOpts{
		Role:        entity.UserRoleAdmin,
		AdminRole:   &role,
		TOTPEnabled: true,
	})
	tok, _ = issueSecurityToken(admin, true)
	rec = securityPOST(e, "/admin/reviews/users/"+buyer.ID.String()+"/reject", tok, `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400", rec.Code)
	}
}

func TestProjectPatchRejectRequiresReason(t *testing.T) {
	var (
		e      *echo.Echo
		gdb    *gorm.DB
		seller *entity.User
		admin  *entity.User
		project   *entity.Project
		tok    string
		role   entity.AdminRole
		rec    *httptest.ResponseRecorder
	)
	e = setupSecurityApp(t)
	gdb = openSecurityDB(t)
	seller = seedSecurityUser(t, gdb, seedOpts{Role: entity.UserRoleSeller})
	project = seedSecurityProject(t, gdb, seller.ID, entity.ProjectStatusPending)
	role = entity.AdminRoleReviewAdmin
	admin = seedSecurityUser(t, gdb, seedOpts{
		Role:        entity.UserRoleAdmin,
		AdminRole:   &role,
		TOTPEnabled: true,
	})
	tok, _ = issueSecurityToken(admin, true)
	rec = securityPATCH(e, "/admin/projects/"+project.ID.String(), tok, `{"status":"CANCELLED"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400", rec.Code)
	}
}

func TestProjectHardDeleteActiveForbidden(t *testing.T) {
	var (
		e      *echo.Echo
		gdb    *gorm.DB
		seller *entity.User
		admin  *entity.User
		project   *entity.Project
		tok    string
		role   entity.AdminRole
		rec    *httptest.ResponseRecorder
	)
	e = setupSecurityApp(t)
	gdb = openSecurityDB(t)
	seller = seedSecurityUser(t, gdb, seedOpts{Role: entity.UserRoleSeller})
	project = seedSecurityProject(t, gdb, seller.ID, entity.ProjectStatusActive)
	role = entity.AdminRoleReviewAdmin
	admin = seedSecurityUser(t, gdb, seedOpts{
		Role:        entity.UserRoleAdmin,
		AdminRole:   &role,
		TOTPEnabled: true,
	})
	tok, _ = issueSecurityToken(admin, true)
	rec = securityDELETE(e, "/admin/projects/"+project.ID.String(), tok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400", rec.Code)
	}
}

func TestProjectHardDeleteDeletedAllowed(t *testing.T) {
	var (
		e      *echo.Echo
		gdb    *gorm.DB
		seller *entity.User
		admin  *entity.User
		project   *entity.Project
		tok    string
		role   entity.AdminRole
		rec    *httptest.ResponseRecorder
		err    error
		n      int64
	)
	e = setupSecurityApp(t)
	gdb = openSecurityDB(t)
	seller = seedSecurityUser(t, gdb, seedOpts{Role: entity.UserRoleSeller})
	project = seedSecurityProject(t, gdb, seller.ID, entity.ProjectStatusDeleted)
	role = entity.AdminRoleReviewAdmin
	admin = seedSecurityUser(t, gdb, seedOpts{
		Role:        entity.UserRoleAdmin,
		AdminRole:   &role,
		TOTPEnabled: true,
	})
	tok, _ = issueSecurityToken(admin, true)
	rec = securityDELETE(e, "/admin/projects/"+project.ID.String(), tok)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d want 204", rec.Code)
	}
	err = gdb.Model(&entity.Project{}).Where("id = ?", project.ID).Count(&n).Error
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("project row count=%d want 0", n)
	}
}

func seedSecurityProject(t *testing.T, gdb *gorm.DB, sellerID uuid.UUID, status entity.ProjectStatus) *entity.Project {
	var (
		now time.Time
		p   *entity.Project
		err error
	)
	t.Helper()
	now = time.Now().UTC()
	p = &entity.Project{
		ID:           uuid.New(),
		SellerID:     sellerID,
		Title:        fmt.Sprintf("project-%s", uuid.New()),
		Description:  "integration test project",
		TargetAmount: 1000,
		DurationType: entity.DurationUnlimited,
		StartDate:    now,
		Status:       status,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err = gdb.Create(p).Error; err != nil {
		t.Fatal(err)
	}
	return p
}

func securityPATCH(e *echo.Echo, path, token, body string) *httptest.ResponseRecorder {
	var (
		rec *httptest.ResponseRecorder
		req *http.Request
	)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	e.ServeHTTP(rec, req)
	return rec
}

func securityDELETE(e *echo.Echo, path, token string) *httptest.ResponseRecorder {
	var (
		rec *httptest.ResponseRecorder
		req *http.Request
	)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	e.ServeHTTP(rec, req)
	return rec
}
