//go:build integration

package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"crowdfund-app/entity"

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
	e = setupApp(t)
	gdb = openDB(t)
	u = seedUser(t, gdb, seedOpts{Role: entity.UserRoleBuyer})
	tok, _ = issueToken(u, true)
	rec = httpPOST(e, "/projects", tok, `{"title":"t","description":"d"}`)
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
	e = setupApp(t)
	gdb = openDB(t)
	role = entity.AdminRoleReviewAdmin
	u = seedUser(t, gdb, seedOpts{
		Role:        entity.UserRoleAdmin,
		AdminRole:   &role,
		TOTPEnabled: true,
	})
	tok, _ = issueToken(u, true)
	rec = httpGET(e, "/admin/projects", tok)
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
	e = setupApp(t)
	gdb = openDB(t)
	role = entity.AdminRoleFinanceAdmin
	u = seedUser(t, gdb, seedOpts{
		Role:        entity.UserRoleAdmin,
		AdminRole:   &role,
		TOTPEnabled: true,
	})
	tok, _ = issueToken(u, true)
	rec = httpGET(e, "/admin/projects", tok)
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
	e = setupApp(t)
	gdb = openDB(t)
	seller = seedUser(t, gdb, seedOpts{Role: entity.UserRoleSeller})
	project = seedProject(t, gdb, seller.ID, entity.ProjectStatusPending)
	rec = httpGET(e, "/projects/"+project.ID.String(), "")
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
	e = setupApp(t)
	gdb = openDB(t)
	seller = seedUser(t, gdb, seedOpts{
		Role:        entity.UserRoleSeller,
		TOTPEnabled: true,
	})
	project = seedProject(t, gdb, seller.ID, entity.ProjectStatusPending)
	tok, _ = issueToken(seller, true)
	rec = httpGET(e, "/projects/"+project.ID.String(), tok)
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
	e = setupApp(t)
	gdb = openDB(t)
	buyer = seedUser(t, gdb, seedOpts{
		Role:          entity.UserRoleBuyer,
		AccountStatus: entity.AccountStatusPending,
	})
	role = entity.AdminRoleReviewAdmin
	admin = seedUser(t, gdb, seedOpts{
		Role:        entity.UserRoleAdmin,
		AdminRole:   &role,
		TOTPEnabled: true,
	})
	tok, _ = issueToken(admin, true)
	rec = httpPOST(e, "/admin/reviews/users/"+buyer.ID.String()+"/reject", tok, `{}`)
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
	e = setupApp(t)
	gdb = openDB(t)
	seller = seedUser(t, gdb, seedOpts{Role: entity.UserRoleSeller})
	project = seedProject(t, gdb, seller.ID, entity.ProjectStatusPending)
	role = entity.AdminRoleReviewAdmin
	admin = seedUser(t, gdb, seedOpts{
		Role:        entity.UserRoleAdmin,
		AdminRole:   &role,
		TOTPEnabled: true,
	})
	tok, _ = issueToken(admin, true)
	rec = httpPATCH(e, "/admin/projects/"+project.ID.String(), tok, `{"status":"CANCELLED"}`)
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
	e = setupApp(t)
	gdb = openDB(t)
	seller = seedUser(t, gdb, seedOpts{Role: entity.UserRoleSeller})
	project = seedProject(t, gdb, seller.ID, entity.ProjectStatusActive)
	role = entity.AdminRoleReviewAdmin
	admin = seedUser(t, gdb, seedOpts{
		Role:        entity.UserRoleAdmin,
		AdminRole:   &role,
		TOTPEnabled: true,
	})
	tok, _ = issueToken(admin, true)
	rec = httpDELETE(e, "/admin/projects/"+project.ID.String(), tok)
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
	e = setupApp(t)
	gdb = openDB(t)
	seller = seedUser(t, gdb, seedOpts{Role: entity.UserRoleSeller})
	project = seedProject(t, gdb, seller.ID, entity.ProjectStatusDeleted)
	role = entity.AdminRoleReviewAdmin
	admin = seedUser(t, gdb, seedOpts{
		Role:        entity.UserRoleAdmin,
		AdminRole:   &role,
		TOTPEnabled: true,
	})
	tok, _ = issueToken(admin, true)
	rec = httpDELETE(e, "/admin/projects/"+project.ID.String(), tok)
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
