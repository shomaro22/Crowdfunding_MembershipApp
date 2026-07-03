//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"crowdfund-app/controller"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type authTokenBody struct {
	AuthToken string `json:"auth_token"`
}

type registerBody struct {
	User struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	} `json:"user"`
}

func TestAuthRegisterLoginRefreshLogout(t *testing.T) {
	var (
		e         *echo.Echo
		email     string
		rec       *httptest.ResponseRecorder
		authTok   string
		refresh   *http.Cookie
		newTok    authTokenBody
		logoutRec *httptest.ResponseRecorder
	)
	e = setupApp(t)
	email = fmt.Sprintf("buyer-%s@test.local", uuid.New())

	rec = httpPOST(e, "/auth/register", "", fmt.Sprintf(
		`{"email":%q,"password":"password123","name":"Integration Buyer"}`,
		email,
	))
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status=%d body=%s", rec.Code, rec.Body.String())
	}
	var reg registerBody
	if err := json.Unmarshal(rec.Body.Bytes(), &reg); err != nil {
		t.Fatal(err)
	}
	if reg.User.Email != email || reg.User.Role != "BUYER" {
		t.Fatalf("register user=%+v", reg.User)
	}

	rec = httpPOST(e, "/auth/login", "", fmt.Sprintf(
		`{"email":%q,"password":"password123"}`,
		email,
	))
	if rec.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &newTok); err != nil || newTok.AuthToken == "" {
		t.Fatalf("login token body=%s", rec.Body.String())
	}
	authTok = newTok.AuthToken
	refresh = findCookie(rec, controller.DefaultRefreshCookie)
	if refresh == nil || refresh.Value == "" {
		t.Fatal("missing refresh_token cookie")
	}

	rec = httpGET(e, "/me", authTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("me status=%d body=%s", rec.Code, rec.Body.String())
	}
	var meBody struct {
		User struct {
			Email string `json:"email"`
		} `json:"user"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &meBody); err != nil {
		t.Fatal(err)
	}
	if meBody.User.Email != email {
		t.Fatalf("me email=%q want %q", meBody.User.Email, email)
	}

	rec = httpPOSTWithCookies(e, "/auth/refresh", "", "", []*http.Cookie{refresh})
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &newTok); err != nil || newTok.AuthToken == "" {
		t.Fatalf("refresh token body=%s", rec.Body.String())
	}
	authTok = newTok.AuthToken
	if rotated := findCookie(rec, controller.DefaultRefreshCookie); rotated != nil && rotated.Value != "" {
		refresh = rotated
	}

	logoutRec = httpPOSTWithCookies(e, "/auth/logout", "", "", []*http.Cookie{refresh})
	if logoutRec.Code != http.StatusOK && logoutRec.Code != http.StatusNoContent {
		t.Fatalf("logout status=%d body=%s", logoutRec.Code, logoutRec.Body.String())
	}

	rec = httpPOSTWithCookies(e, "/auth/refresh", "", "", []*http.Cookie{refresh})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("refresh after logout status=%d want 204", rec.Code)
	}
	if rec.Body.Len() > 0 {
		t.Fatalf("refresh after logout body=%s want empty", rec.Body.String())
	}
}

func TestAuthLoginInvalidCredentials(t *testing.T) {
	var (
		e   *echo.Echo
		rec *httptest.ResponseRecorder
	)
	e = setupApp(t)
	rec = httpPOST(e, "/auth/login", "", `{"email":"nobody@test.local","password":"wrong"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", rec.Code)
	}
	if bodyCode(rec) != "INVALID_CREDENTIALS" {
		t.Fatalf("code=%q", bodyCode(rec))
	}
}

func TestAuthRegisterDuplicateEmail(t *testing.T) {
	var (
		e     *echo.Echo
		email string
		rec   *httptest.ResponseRecorder
	)
	e = setupApp(t)
	email = fmt.Sprintf("dup-%s@test.local", uuid.New())
	body := fmt.Sprintf(`{"email":%q,"password":"password123"}`, email)
	rec = httpPOST(e, "/auth/register", "", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("first register status=%d", rec.Code)
	}
	rec = httpPOST(e, "/auth/register", "", body)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate status=%d want 409", rec.Code)
	}
}

func findCookie(rec *httptest.ResponseRecorder, name string) *http.Cookie {
	var i int
	for i = range rec.Result().Cookies() {
		if rec.Result().Cookies()[i].Name == name {
			return rec.Result().Cookies()[i]
		}
	}
	return nil
}

func httpPOSTWithCookies(e *echo.Echo, path, token, body string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	var (
		rec *httptest.ResponseRecorder
		req *http.Request
	)
	rec = httptest.NewRecorder()
	if body == "" {
		req = httptest.NewRequest(http.MethodPost, path, nil)
	} else {
		req = httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	e.ServeHTTP(rec, req)
	return rec
}
