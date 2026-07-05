//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"crowdfund-app/model"
	"crowdfund-app/usecase"

	"github.com/labstack/echo/v4"
	"github.com/pquerna/otp/totp"
)

const testTOTPSecret = "JBSWY3DPEHPK3PXP"

func seedSellerWithTOTP(t *testing.T) *model.User {
	var (
		gdb    = testDB(t)
		hash   string
		enc    string
		now    time.Time
		u      *model.User
		err    error
	)

	t.Helper()
	hash, err = usecase.HashPassword("password123")
	if err != nil {
		t.Fatal(err)
	}
	enc, err = usecase.EncryptSecret([]byte(testJWTSecret), testTOTPSecret)
	if err != nil {
		t.Fatal(err)
	}
	now = time.Now().UTC()
	u = &model.User{
		Email:         fmt.Sprintf("seller-2fa-%d@test.local", now.UnixNano()),
		PasswordHash:  hash,
		Role:          model.UserRoleSeller,
		AccountStatus: model.AccountStatusActive,
		SellerStatus:  model.SellerStatusActive,
		TOTPEnabled:   true,
		TOTPSecret:    &enc,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err = gdb.Create(u).Error; err != nil {
		t.Fatal(err)
	}
	return u
}

func TestLogin2FAChallengeWithPendingTokenBody(t *testing.T) {
	var (
		e       = setupApp(t)
		u       = seedSellerWithTOTP(t)
		code    string
		login   login2FAResponse
		rec     = httpPOST(e, "/auth/login", "", fmt.Sprintf(
			`{"email":%q,"password":"password123"}`, u.Email,
		))
		err     error
	)

	if rec.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err = json.Unmarshal(rec.Body.Bytes(), &login); err != nil {
		t.Fatal(err)
	}
	if !login.Requires2FA || login.PendingToken == "" {
		t.Fatalf("login response=%+v", login)
	}

	code, err = totp.GenerateCode(testTOTPSecret, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	rec = httpPOST(e, "/auth/2fa/challenge", "", fmt.Sprintf(
		`{"code":%q,"pending_token":%q}`, code, login.PendingToken,
	))
	if rec.Code != http.StatusOK {
		t.Fatalf("challenge status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestLogin2FAChallengePrefersBodyOverStaleCookie(t *testing.T) {
	var (
		e     = setupApp(t)
		u     = seedSellerWithTOTP(t)
		code  string
		login login2FAResponse
		rec   = httpPOST(e, "/auth/login", "", fmt.Sprintf(
			`{"email":%q,"password":"password123"}`, u.Email,
		))
		err error
	)

	if rec.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err = json.Unmarshal(rec.Body.Bytes(), &login); err != nil {
		t.Fatal(err)
	}
	if !login.Requires2FA || login.PendingToken == "" {
		t.Fatalf("login response=%+v", login)
	}

	code, err = totp.GenerateCode(testTOTPSecret, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	rec = httpPOSTWithCookie(e, "/auth/2fa/challenge", "", fmt.Sprintf(
		`{"code":%q,"pending_token":%q}`, code, login.PendingToken,
	), "2fa_pending=expired-stale-token")
	if rec.Code != http.StatusOK {
		t.Fatalf("challenge status=%d body=%s", rec.Code, rec.Body.String())
	}
}

type login2FAResponse struct {
	Requires2FA  bool   `json:"requires_2fa"`
	PendingToken string `json:"pending_token"`
}

func httpPOSTWithCookie(e *echo.Echo, path, token, body, cookieHeader string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if cookieHeader != "" {
		req.Header.Set("Cookie", cookieHeader)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	e.ServeHTTP(rec, req)
	return rec
}
