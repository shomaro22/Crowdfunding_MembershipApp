//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"crowdfund-app/model"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// §15.5 セキュリティテスト（認証フェーズ）。DATABASE_URL 必須。Rate Limit テストは REDIS_ADDR 必須。

func TestSecurityNoJWT(t *testing.T) {
	var (
		e   *echo.Echo
		rec *httptest.ResponseRecorder
	)
	e = setupApp(t)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/me", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", rec.Code)
	}
	if bodyCode(rec) != "UNAUTHORIZED" {
		t.Fatalf("code=%q", bodyCode(rec))
	}
}

func TestSecurityTamperedJWT(t *testing.T) {
	var (
		e     *echo.Echo
		gdb   *gorm.DB
		u     *model.User
		tok   string
		rec   *httptest.ResponseRecorder
		req   *http.Request
	)
	e = setupApp(t)
	gdb = testDB(t)
	u = seedUser(t, gdb, seedOpts{Role: model.UserRoleBuyer})
	tok, _ = issueToken(u, true)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+tok+"tampered")
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", rec.Code)
	}
}

func TestSecurityExpiredJWT(t *testing.T) {
	var (
		e   *echo.Echo
		gdb *gorm.DB
		u   *model.User
		tok string
		rec *httptest.ResponseRecorder
		req *http.Request
	)
	e = setupApp(t)
	gdb = testDB(t)
	u = seedUser(t, gdb, seedOpts{Role: model.UserRoleBuyer})
	tok = expiredToken(u.ID)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", rec.Code)
	}
}

func TestSecurityRevokedToken(t *testing.T) {
	var (
		e   *echo.Echo
		gdb *gorm.DB
		u   *model.User
		tok string
		rec *httptest.ResponseRecorder
		req *http.Request
	)
	e = setupApp(t)
	gdb = testDB(t)
	u = seedUser(t, gdb, seedOpts{Role: model.UserRoleBuyer})
	tok, _ = issueToken(u, true)
	if err := gdb.Model(&model.User{}).Where("id = ?", u.ID).
		UpdateColumn("token_version", gorm.Expr("token_version + 1")).Error; err != nil {
		t.Fatal(err)
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", rec.Code)
	}
	if bodyCode(rec) != "TOKEN_REVOKED" {
		t.Fatalf("code=%q want TOKEN_REVOKED", bodyCode(rec))
	}
}

func TestSecurityBuyerCannotAccessAdmin(t *testing.T) {
	var (
		e   *echo.Echo
		gdb *gorm.DB
		u   *model.User
		tok string
		rec *httptest.ResponseRecorder
	)
	e = setupApp(t)
	gdb = testDB(t)
	u = seedUser(t, gdb, seedOpts{Role: model.UserRoleBuyer})
	tok, _ = issueToken(u, true)
	rec = httpGET(e, "/admin/reviews/users", tok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rec.Code)
	}
}

func TestSecurityAdminRequires2FA(t *testing.T) {
	var (
		e     *echo.Echo
		gdb   *gorm.DB
		u     *model.User
		tok   string
		role  model.AdminRole
		rec   *httptest.ResponseRecorder
	)
	e = setupApp(t)
	gdb = testDB(t)
	role = model.AdminRoleReviewAdmin
	u = seedUser(t, gdb, seedOpts{
		Role:      model.UserRoleAdmin,
		AdminRole: &role,
		TOTPEnabled: true,
	})
	tok, _ = issueToken(u, false)
	rec = httpGET(e, "/admin/reviews/users", tok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rec.Code)
	}
	if bodyCode(rec) != "2FA_REQUIRED" {
		t.Fatalf("code=%q want 2FA_REQUIRED", bodyCode(rec))
	}
}

func TestSecuritySellerWriteRequires2FA(t *testing.T) {
	var (
		e   *echo.Echo
		gdb *gorm.DB
		u   *model.User
		tok string
		rec *httptest.ResponseRecorder
	)
	e = setupApp(t)
	gdb = testDB(t)
	u = seedUser(t, gdb, seedOpts{
		Role:        model.UserRoleSeller,
		TOTPEnabled: true,
	})
	tok, _ = issueToken(u, false)
	rec = httpPOST(e, "/projects", tok, `{"title":"t","description":"d"}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rec.Code)
	}
	if bodyCode(rec) != "2FA_REQUIRED" {
		t.Fatalf("code=%q want 2FA_REQUIRED", bodyCode(rec))
	}
}

func TestSecurityChangePasswordRequiresTOTP(t *testing.T) {
	var (
		e   *echo.Echo
		gdb *gorm.DB
		u   *model.User
		tok string
		rec *httptest.ResponseRecorder
	)
	e = setupApp(t)
	gdb = testDB(t)
	u = seedUser(t, gdb, seedOpts{
		Role:        model.UserRoleSeller,
		TOTPEnabled: true,
	})
	tok, _ = issueToken(u, true)
	rec = httpPOST(e, "/me/change-password", tok,
		`{"current_password":"password123","new_password":"newpassword123"}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rec.Code)
	}
	if bodyCode(rec) != "2FA_REQUIRED" {
		t.Fatalf("code=%q want 2FA_REQUIRED", bodyCode(rec))
	}
}

func TestBodyCodeParsesRateLimitResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Body.WriteString(`{"error":"rate limit exceeded","code":"RATE_LIMITED","retry_after_seconds":12}`)
	if bodyCode(rec) != "RATE_LIMITED" {
		t.Fatalf("code=%q want RATE_LIMITED", bodyCode(rec))
	}
}

func TestSecurityRateLimitLogin(t *testing.T) {
	var (
		e     *echo.Echo
		email string
		i     int
		rec   *httptest.ResponseRecorder
	)
	if strings.TrimSpace(os.Getenv("REDIS_ADDR")) == "" {
		t.Skip("REDIS_ADDR が未設定のためスキップ")
	}
	e = setupApp(t)
	email = fmt.Sprintf("rl-%s@test.local", uuid.New())
	for i = 0; i < 5; i++ {
		rec = httpPOST(e, "/auth/login", "", fmt.Sprintf(
			`{"email":%q,"password":"wrong"}`, email))
		if rec.Code == http.StatusTooManyRequests {
			t.Fatalf("rate limited too early at attempt %d", i+1)
		}
	}
	rec = httpPOST(e, "/auth/login", "", fmt.Sprintf(
		`{"email":%q,"password":"wrong"}`, email))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status=%d want 429", rec.Code)
	}
	if bodyCode(rec) != "RATE_LIMITED" {
		t.Fatalf("code=%q want RATE_LIMITED", bodyCode(rec))
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("missing Retry-After header")
	}
}
