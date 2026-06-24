package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"crowdfund-app/auth"
	"crowdfund-app/db"
	"crowdfund-app/entity"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// §15.5 セキュリティテスト（認証フェーズ）。DATABASE_URL 必須。Rate Limit テストは REDIS_ADDR 必須。
const securityTestJWTSecret = "01234567890123456789012345678901"

func TestSecurityNoJWT(t *testing.T) {
	var (
		e   *echo.Echo
		rec *httptest.ResponseRecorder
	)
	e = setupSecurityApp(t)
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
		u     *entity.User
		tok   string
		rec   *httptest.ResponseRecorder
		req   *http.Request
	)
	e = setupSecurityApp(t)
	gdb = openSecurityDB(t)
	u = seedSecurityUser(t, gdb, seedOpts{Role: entity.UserRoleBuyer})
	tok, _ = issueSecurityToken(u, true)
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
		u   *entity.User
		tok string
		rec *httptest.ResponseRecorder
		req *http.Request
	)
	e = setupSecurityApp(t)
	gdb = openSecurityDB(t)
	u = seedSecurityUser(t, gdb, seedOpts{Role: entity.UserRoleBuyer})
	tok = expiredSecurityToken(u.ID)
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
		u   *entity.User
		tok string
		rec *httptest.ResponseRecorder
		req *http.Request
	)
	e = setupSecurityApp(t)
	gdb = openSecurityDB(t)
	u = seedSecurityUser(t, gdb, seedOpts{Role: entity.UserRoleBuyer})
	tok, _ = issueSecurityToken(u, true)
	if err := gdb.Model(&entity.User{}).Where("id = ?", u.ID).
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
		u   *entity.User
		tok string
		rec *httptest.ResponseRecorder
	)
	e = setupSecurityApp(t)
	gdb = openSecurityDB(t)
	u = seedSecurityUser(t, gdb, seedOpts{Role: entity.UserRoleBuyer})
	tok, _ = issueSecurityToken(u, true)
	rec = securityGET(e, "/admin/reviews/users", tok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rec.Code)
	}
}

func TestSecurityAdminRequires2FA(t *testing.T) {
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
		Role:      entity.UserRoleAdmin,
		AdminRole: &role,
		TOTPEnabled: true,
	})
	tok, _ = issueSecurityToken(u, false)
	rec = securityGET(e, "/admin/reviews/users", tok)
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
		u   *entity.User
		tok string
		rec *httptest.ResponseRecorder
	)
	e = setupSecurityApp(t)
	gdb = openSecurityDB(t)
	u = seedSecurityUser(t, gdb, seedOpts{
		Role:        entity.UserRoleSeller,
		TOTPEnabled: true,
	})
	tok, _ = issueSecurityToken(u, false)
	rec = securityPOST(e, "/projects", tok, `{"title":"t","description":"d"}`)
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
		u   *entity.User
		tok string
		rec *httptest.ResponseRecorder
	)
	e = setupSecurityApp(t)
	gdb = openSecurityDB(t)
	u = seedSecurityUser(t, gdb, seedOpts{
		Role:        entity.UserRoleSeller,
		TOTPEnabled: true,
	})
	tok, _ = issueSecurityToken(u, true)
	rec = securityPOST(e, "/me/change-password", tok,
		`{"current_password":"password123","new_password":"newpassword123"}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rec.Code)
	}
	if bodyCode(rec) != "2FA_REQUIRED" {
		t.Fatalf("code=%q want 2FA_REQUIRED", bodyCode(rec))
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
	e = setupSecurityApp(t)
	email = fmt.Sprintf("rl-%s@test.local", uuid.New())
	for i = 0; i < 5; i++ {
		rec = securityPOST(e, "/auth/login", "", fmt.Sprintf(
			`{"email":%q,"password":"wrong"}`, email))
		if rec.Code == http.StatusTooManyRequests {
			t.Fatalf("rate limited too early at attempt %d", i+1)
		}
	}
	rec = securityPOST(e, "/auth/login", "", fmt.Sprintf(
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

type seedOpts struct {
	Role        entity.UserRole
	AdminRole   *entity.AdminRole
	TOTPEnabled bool
}

func setupSecurityApp(t *testing.T) *echo.Echo {
	var (
		e     *echo.Echo
		gdb   *gorm.DB
		sqlDB *sql.DB
		err   error
	)
	t.Helper()
	if strings.TrimSpace(os.Getenv("DATABASE_URL")) == "" {
		t.Skip("DATABASE_URL が未設定のためスキップ")
	}
	gdb, err = db.Open(os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Skip("DB 接続不可のためスキップ: ", err)
	}
	sqlDB, err = gdb.DB()
	if err != nil {
		t.Skip("DB 接続不可のためスキップ: ", err)
	}
	if err = sqlDB.Ping(); err != nil {
		t.Skip("DB 接続不可のためスキップ: ", err)
	}
	_ = os.Setenv("JWT_SECRET", securityTestJWTSecret)
	e, _, err = buildApp()
	if err != nil {
		t.Fatalf("buildApp: %v", err)
	}
	return e
}

func openSecurityDB(t *testing.T) *gorm.DB {
	var (
		gdb *gorm.DB
		err error
	)
	t.Helper()
	gdb, err = db.Open(os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	return gdb
}

func seedSecurityUser(t *testing.T, gdb *gorm.DB, opts seedOpts) *entity.User {
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
		Email:         fmt.Sprintf("%s-%s@test.local", opts.Role, uuid.New()),
		PasswordHash:  hash,
		Role:          opts.Role,
		AccountStatus: entity.AccountStatusActive,
		SellerStatus:  entity.SellerStatusNotApplicable,
		AdminRole:     opts.AdminRole,
		TOTPEnabled:   opts.TOTPEnabled,
		TokenVersion:  0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if opts.Role == entity.UserRoleSeller {
		u.SellerStatus = entity.SellerStatusActive
	}
	if err = gdb.Create(u).Error; err != nil {
		t.Fatal(err)
	}
	return u
}

func issueSecurityToken(u *entity.User, twoFA bool) (string, error) {
	return auth.NewTokenService([]byte(securityTestJWTSecret), auth.DefaultAuthTTL).Issue(auth.AuthClaims{
		UserID:        u.ID,
		Role:          u.Role,
		TwoFAVerified: twoFA,
		AdminRole:     u.AdminRole,
		TokenVersion:  u.TokenVersion,
	})
}

func expiredSecurityToken(userID uuid.UUID) string {
	var (
		tok *jwt.Token
		s   string
		err error
	)
	tok = jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":          userID.String(),
		"role":         string(entity.UserRoleBuyer),
		"2fa_verified": false,
		"tv":           float64(0),
		"exp":          time.Now().Add(-time.Hour).Unix(),
		"iat":          time.Now().Add(-2 * time.Hour).Unix(),
	})
	s, err = tok.SignedString([]byte(securityTestJWTSecret))
	if err != nil {
		return ""
	}
	return s
}

func securityGET(e *echo.Echo, path, token string) *httptest.ResponseRecorder {
	var (
		rec *httptest.ResponseRecorder
		req *http.Request
	)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	e.ServeHTTP(rec, req)
	return rec
}

func securityPOST(e *echo.Echo, path, token, body string) *httptest.ResponseRecorder {
	var (
		rec *httptest.ResponseRecorder
		req *http.Request
	)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	e.ServeHTTP(rec, req)
	return rec
}

func bodyCode(rec *httptest.ResponseRecorder) string {
	var (
		m   map[string]string
		raw string
	)
	raw = strings.TrimSpace(rec.Body.String())
	if raw == "" {
		return ""
	}
	if json.Unmarshal([]byte(raw), &m) != nil {
		return ""
	}
	return m["code"]
}
