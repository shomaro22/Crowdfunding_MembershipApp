//go:build integration

package integration

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"crowdfund-app/auth"
	"crowdfund-app/bootstrap"
	"crowdfund-app/db"
	"crowdfund-app/model"
	"crowdfund-app/stripe"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const testJWTSecret = "01234567890123456789012345678901"

var (
	testGDB *gorm.DB
	testRDB *redis.Client
)

type seedOpts struct {
	Role          model.UserRole
	AdminRole     *model.AdminRole
	TOTPEnabled   bool
	AccountStatus model.AccountStatus
}

func TestMain(m *testing.M) {
	var (
		code int
		err  error
	)
	if strings.TrimSpace(os.Getenv("DATABASE_URL")) != "" {
		err = initTestInfra()
		if err != nil {
			fmt.Fprintf(os.Stderr, "integration infra: %v\n", err)
			os.Exit(1)
		}
		defer closeTestInfra()
	}
	code = m.Run()
	os.Exit(code)
}

func initTestInfra() error {
	var (
		sqlDB *sql.DB
		err   error
	)
	err = db.Up(os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	testGDB, err = db.Open(os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	sqlDB, err = testGDB.DB()
	if err != nil {
		return err
	}
	err = sqlDB.Ping()
	if err != nil {
		return err
	}
	testRDB = db.OpenRedis()
	return nil
}

func closeTestInfra() {
	var sqlDB *sql.DB
	if testGDB != nil {
		sqlDB, _ = testGDB.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
		testGDB = nil
	}
	if testRDB != nil {
		_ = testRDB.Close()
		testRDB = nil
	}
}

func appOpts(stripeClient stripe.Client) *bootstrap.Options {
	return &bootstrap.Options{Stripe: stripeClient, DB: testGDB, Redis: testRDB}
}

func setupApp(t *testing.T) *echo.Echo {
	return setupAppWithOpts(t, nil)
}

func setupAppWithOpts(t *testing.T, opts *bootstrap.Options) *echo.Echo {
	var (
		e   *echo.Echo
		err error
	)
	t.Helper()
	if testGDB == nil {
		t.Skip("DATABASE_URL が未設定のためスキップ")
	}
	_ = os.Setenv("JWT_SECRET", testJWTSecret)
	if opts == nil {
		opts = appOpts(nil)
	}
	e, _, err = bootstrap.BuildHTTP(opts)
	if err != nil {
		t.Fatalf("BuildHTTP: %v", err)
	}
	return e
}

func setupPaymentApp(t *testing.T) (*echo.Echo, *stripe.StubClient) {
	var stub *stripe.StubClient
	t.Helper()
	stub = stripe.NewStubClient()
	return setupAppWithOpts(t, appOpts(stub)), stub
}

func openDB(t *testing.T) *gorm.DB {
	t.Helper()
	if testGDB == nil {
		t.Fatal("DATABASE_URL が未設定")
	}
	return testGDB
}

func seedUser(t *testing.T, gdb *gorm.DB, opts seedOpts) *model.User {
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
		Email:         fmt.Sprintf("%s-%s@test.local", opts.Role, uuid.New()),
		PasswordHash:  hash,
		Role:          opts.Role,
		AccountStatus: model.AccountStatusActive,
		SellerStatus:  model.SellerStatusNotApplicable,
		AdminRole:     opts.AdminRole,
		TOTPEnabled:   opts.TOTPEnabled,
		TokenVersion:  0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if opts.AccountStatus != "" {
		u.AccountStatus = opts.AccountStatus
	}
	if opts.Role == model.UserRoleSeller {
		u.SellerStatus = model.SellerStatusActive
	}
	if err = gdb.Create(u).Error; err != nil {
		t.Fatal(err)
	}
	return u
}

func seedProject(t *testing.T, gdb *gorm.DB, sellerID uuid.UUID, status model.ProjectStatus) *model.Project {
	var (
		now time.Time
		p   *model.Project
		err error
	)
	t.Helper()
	now = time.Now().UTC()
	p = &model.Project{
		ID:           uuid.New(),
		SellerID:     sellerID,
		Title:        fmt.Sprintf("project-%s", uuid.New()),
		Description:  "integration test project",
		TargetAmount: 1000,
		DurationType: model.DurationUnlimited,
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

func issueToken(u *model.User, twoFA bool) (string, error) {
	return auth.NewTokenService([]byte(testJWTSecret), auth.DefaultAuthTTL).Issue(auth.AuthClaims{
		UserID: u.ID, Role: u.Role, TwoFAVerified: twoFA,
		AdminRole: u.AdminRole, TokenVersion: u.TokenVersion,
	})
}

func issueAdminToken(t *testing.T, gdb *gorm.DB, role model.AdminRole) string {
	var (
		u   *model.User
		tok string
		err error
	)
	t.Helper()
	u = seedUser(t, gdb, seedOpts{Role: model.UserRoleAdmin, AdminRole: &role, TOTPEnabled: true})
	tok, err = issueToken(u, true)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func httpGET(e *echo.Echo, path, token string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	e.ServeHTTP(rec, req)
	return rec
}

func httpPOST(e *echo.Echo, path, token, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	e.ServeHTTP(rec, req)
	return rec
}

func httpPUT(e *echo.Echo, path, token, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	e.ServeHTTP(rec, req)
	return rec
}

func httpPATCH(e *echo.Echo, path, token, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	e.ServeHTTP(rec, req)
	return rec
}

func httpDELETE(e *echo.Echo, path, token string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	e.ServeHTTP(rec, req)
	return rec
}

func bodyCode(rec *httptest.ResponseRecorder) string {
	var m map[string]any
	raw := strings.TrimSpace(rec.Body.String())
	if raw == "" || json.Unmarshal([]byte(raw), &m) != nil {
		return ""
	}
	c, _ := m["code"].(string)
	return c
}

func mediaUploadPOST(e *echo.Echo, token string, data []byte, filename string) *httptest.ResponseRecorder {
	var (
		body   bytes.Buffer
		writer *multipart.Writer
		part   io.Writer
		rec    *httptest.ResponseRecorder
		req    *http.Request
		err    error
	)
	writer = multipart.NewWriter(&body)
	part, err = writer.CreateFormFile("file", filename)
	if err != nil {
		panic(err)
	}
	_, _ = part.Write(data)
	_ = writer.Close()
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/media/images", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	e.ServeHTTP(rec, req)
	return rec
}

func expiredToken(userID uuid.UUID) string {
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID.String(), "role": string(model.UserRoleBuyer),
		"2fa_verified": false, "tv": float64(0),
		"exp": time.Now().Add(-time.Hour).Unix(),
		"iat": time.Now().Add(-2 * time.Hour).Unix(),
	})
	s, err := tok.SignedString([]byte(testJWTSecret))
	if err != nil {
		return ""
	}
	return s
}
