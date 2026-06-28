//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func TestOperationsNoAuthReturns401(t *testing.T) {
	var (
		e   *echo.Echo
		rec *httptest.ResponseRecorder
	)
	e = setupApp(t)
	rec = httpGET(e, "/admin/activity", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", rec.Code)
	}
}

func TestOperationsBuyerCannotAccessActivity(t *testing.T) {
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
	rec = httpGET(e, "/admin/activity", tok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rec.Code)
	}
}

func TestOperationsRequires2FA(t *testing.T) {
	var (
		e    *echo.Echo
		gdb  *gorm.DB
		u    *entity.User
		tok  string
		role entity.AdminRole
		rec  *httptest.ResponseRecorder
	)
	e = setupApp(t)
	gdb = openDB(t)
	role = entity.AdminRoleSuperAdmin
	u = seedUser(t, gdb, seedOpts{
		Role:        entity.UserRoleAdmin,
		AdminRole:   &role,
		TOTPEnabled: true,
	})
	tok, _ = issueToken(u, false)
	rec = httpGET(e, "/admin/activity", tok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rec.Code)
	}
	if bodyCode(rec) != "2FA_REQUIRED" {
		t.Fatalf("code=%q want 2FA_REQUIRED", bodyCode(rec))
	}
}

func TestOperationsSuperAdminCanAccessAll(t *testing.T) {
	var (
		e    *echo.Echo
		gdb  *gorm.DB
		tok  string
		path string
		rec  *httptest.ResponseRecorder
	)
	e = setupApp(t)
	gdb = openDB(t)
	tok = issueAdminToken(t, gdb, entity.AdminRoleSuperAdmin)

	for _, path = range []string{
		"/admin/activity",
		"/admin/security",
		"/admin/batches",
		"/admin/audit-logs",
	} {
		rec = httpGET(e, path, tok)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status=%d want 200 body=%s", path, rec.Code, rec.Body.String())
		}
	}

	rec = httpPOST(e, "/admin/batches/purge_tokens/run?dry_run=true", tok, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("POST batch run status=%d want 200 body=%s", rec.Code, rec.Body.String())
	}
}

func TestOperationsFinanceAdminRBAC(t *testing.T) {
	var (
		e    *echo.Echo
		gdb  *gorm.DB
		tok  string
		rec  *httptest.ResponseRecorder
	)
	e = setupApp(t)
	gdb = openDB(t)
	tok = issueAdminToken(t, gdb, entity.AdminRoleFinanceAdmin)

	for _, path := range []string{"/admin/activity", "/admin/batches"} {
		rec = httpGET(e, path, tok)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status=%d want 200", path, rec.Code)
		}
	}

	for _, path := range []string{"/admin/security", "/admin/audit-logs"} {
		rec = httpGET(e, path, tok)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("GET %s status=%d want 403", path, rec.Code)
		}
	}

	rec = httpPOST(e, "/admin/batches/purge_tokens/run?dry_run=true", tok, "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("POST batch run status=%d want 403", rec.Code)
	}
}

func TestOperationsSupportAdminRBAC(t *testing.T) {
	var (
		e   *echo.Echo
		gdb *gorm.DB
		tok string
		rec *httptest.ResponseRecorder
	)
	e = setupApp(t)
	gdb = openDB(t)
	tok = issueAdminToken(t, gdb, entity.AdminRoleSupportAdmin)

	for _, path := range []string{"/admin/activity", "/admin/security"} {
		rec = httpGET(e, path, tok)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status=%d want 200", path, rec.Code)
		}
	}

	for _, path := range []string{"/admin/batches", "/admin/audit-logs"} {
		rec = httpGET(e, path, tok)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("GET %s status=%d want 403", path, rec.Code)
		}
	}

	rec = httpPOST(e, "/admin/batches/purge_tokens/run?dry_run=true", tok, "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("POST batch run status=%d want 403", rec.Code)
	}
}

func TestOperationsReviewAdminDenied(t *testing.T) {
	var (
		e    *echo.Echo
		gdb  *gorm.DB
		tok  string
		path string
		rec  *httptest.ResponseRecorder
	)
	e = setupApp(t)
	gdb = openDB(t)
	tok = issueAdminToken(t, gdb, entity.AdminRoleReviewAdmin)

	for _, path = range []string{
		"/admin/activity",
		"/admin/security",
		"/admin/batches",
		"/admin/audit-logs",
	} {
		rec = httpGET(e, path, tok)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("GET %s status=%d want 403", path, rec.Code)
		}
	}
}

func TestOperationsListActivityReturnsSeededLog(t *testing.T) {
	var (
		e      *echo.Echo
		gdb    *gorm.DB
		tok    string
		logID  uuid.UUID
		rec    *httptest.ResponseRecorder
		body   string
	)
	e = setupApp(t)
	gdb = openDB(t)
	tok = issueAdminToken(t, gdb, entity.AdminRoleSuperAdmin)
	logID = seedOpsActivityLog(t, gdb, entity.UserLoginFail, strPtr("203.0.113.55"), time.Now().UTC())

	rec = httpGET(e, "/admin/activity?action=user.login_failed&limit=50", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want 200", rec.Code)
	}
	body = rec.Body.String()
	if !strings.Contains(body, logID.String()) {
		t.Fatalf("response missing log id %s: %s", logID, body)
	}
}

func TestOperationsSecurityEventsCounts(t *testing.T) {
	var (
		e     *echo.Echo
		gdb   *gorm.DB
		tok   string
		rec   *httptest.ResponseRecorder
		m     map[string]any
		now   time.Time
	)
	e = setupApp(t)
	gdb = openDB(t)
	tok = issueAdminToken(t, gdb, entity.AdminRoleSupportAdmin)
	now = time.Now().UTC()
	seedOpsActivityLog(t, gdb, entity.UserLoginFail, strPtr("203.0.113.1"), now)
	seedOpsActivityLog(t, gdb, entity.SecRateLimit, strPtr("203.0.113.2"), now)

	rec = httpGET(e, "/admin/security", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want 200", rec.Code)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if n, _ := m["login_failed_count"].(float64); n < 1 {
		t.Fatalf("login_failed_count=%v want >= 1", m["login_failed_count"])
	}
	if n, _ := m["rate_limited_count"].(float64); n < 1 {
		t.Fatalf("rate_limited_count=%v want >= 1", m["rate_limited_count"])
	}
}

func TestOperationsSuperAdminListAuditLogs(t *testing.T) {
	var (
		e       *echo.Echo
		gdb     *gorm.DB
		admin   *entity.User
		tok     string
		auditID uuid.UUID
		rec     *httptest.ResponseRecorder
		body    string
	)
	e = setupApp(t)
	gdb = openDB(t)
	admin = seedUser(t, gdb, seedOpts{
		Role:        entity.UserRoleAdmin,
		AdminRole:   adminRolePtr(entity.AdminRoleSuperAdmin),
		TOTPEnabled: true,
	})
	tok, _ = issueToken(admin, true)
	auditID = seedOpsAuditLog(t, gdb, admin.ID, "batch.manual_run")

	rec = httpGET(e, "/admin/audit-logs?action=batch.manual_run", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want 200", rec.Code)
	}
	body = rec.Body.String()
	if !strings.Contains(body, auditID.String()) {
		t.Fatalf("response missing audit id %s: %s", auditID, body)
	}
}

func TestOperationsRunBatchInvalidJob(t *testing.T) {
	var (
		e   *echo.Echo
		gdb *gorm.DB
		tok string
		rec *httptest.ResponseRecorder
	)
	e = setupApp(t)
	gdb = openDB(t)
	tok = issueAdminToken(t, gdb, entity.AdminRoleSuperAdmin)

	rec = httpPOST(e, "/admin/batches/not_a_job/run?dry_run=true", tok, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", rec.Code, rec.Body.String())
	}
}

func seedOpsActivityLog(t *testing.T, gdb *gorm.DB, action entity.ActivityAction, ip *string, at time.Time) uuid.UUID {
	var (
		id  uuid.UUID
		row *entity.ActivityLog
		err error
	)
	t.Helper()
	id = uuid.New()
	row = &entity.ActivityLog{
		ID:        id,
		Action:    action,
		IPAddress: ip,
		CreatedAt: at,
	}
	if err = gdb.Create(row).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = gdb.Delete(&entity.ActivityLog{}, "id = ?", id).Error
	})
	return id
}

func seedOpsAuditLog(t *testing.T, gdb *gorm.DB, adminID uuid.UUID, action string) uuid.UUID {
	var (
		id  uuid.UUID
		row *entity.AuditLog
		err error
	)
	t.Helper()
	id = uuid.New()
	row = &entity.AuditLog{
		ID:         id,
		AdminID:    adminID,
		Action:     action,
		Resource:   "batch",
		ResourceID: "purge_tokens",
		CreatedAt:  time.Now().UTC(),
	}
	if err = gdb.Create(row).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = gdb.Delete(&entity.AuditLog{}, "id = ?", id).Error
	})
	return id
}

func adminRolePtr(r entity.AdminRole) *entity.AdminRole {
	return &r
}

func strPtr(s string) *string {
	return &s
}
