//go:build integration

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

const securityTestJWTSecret = "01234567890123456789012345678901"

type seedOpts struct {
	Role          entity.UserRole
	AdminRole     *entity.AdminRole
	TOTPEnabled   bool
	AccountStatus entity.AccountStatus
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
	e, _, err = buildApp(nil)
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
	if opts.AccountStatus != "" {
		u.AccountStatus = opts.AccountStatus
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
		m   map[string]any
		raw string
	)
	raw = strings.TrimSpace(rec.Body.String())
	if raw == "" {
		return ""
	}
	if json.Unmarshal([]byte(raw), &m) != nil {
		return ""
	}
	c, _ := m["code"].(string)
	return c
}
