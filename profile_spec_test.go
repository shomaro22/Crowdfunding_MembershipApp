//go:build integration

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"crowdfund-app/entity"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func TestProfileUnauthorized(t *testing.T) {
	var (
		e   *echo.Echo
		rec *httptest.ResponseRecorder
	)
	e = setupSecurityApp(t)
	rec = securityGET(e, "/me/profile", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("get status=%d want 401", rec.Code)
	}
	rec = securityPUT(e, "/me/profile", "", `{"name":"Alice"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("put status=%d want 401", rec.Code)
	}
}

func TestProfileGetAndUpdate(t *testing.T) {
	var (
		e      *echo.Echo
		gdb    *gorm.DB
		buyer  *entity.User
		tok    string
		rec    *httptest.ResponseRecorder
		body   map[string]any
	)
	e = setupSecurityApp(t)
	gdb = openSecurityDB(t)
	buyer = seedSecurityUser(t, gdb, seedOpts{Role: entity.UserRoleBuyer})
	tok, _ = issueSecurityToken(buyer, true)

	rec = securityGET(e, "/me/profile", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["email"] != buyer.Email {
		t.Fatalf("email=%v want %s", body["email"], buyer.Email)
	}

	rec = securityPUT(e, "/me/profile", tok, `{"name":"Alice","avatar_url":"https://example.com/a.png"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("put status=%d body=%s", rec.Code, rec.Body.String())
	}
	body = nil
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["name"] != "Alice" {
		t.Fatalf("name=%v want Alice", body["name"])
	}
	if body["avatar_url"] != "https://example.com/a.png" {
		t.Fatalf("avatar_url=%v", body["avatar_url"])
	}

	rec = securityGET(e, "/me/profile", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("get after update status=%d body=%s", rec.Code, rec.Body.String())
	}
	body = nil
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["name"] != "Alice" {
		t.Fatalf("persisted name=%v want Alice", body["name"])
	}
}

func TestProfileUpdateBadRequest(t *testing.T) {
	var (
		e     *echo.Echo
		gdb   *gorm.DB
		buyer *entity.User
		tok   string
		rec   *httptest.ResponseRecorder
	)
	e = setupSecurityApp(t)
	gdb = openSecurityDB(t)
	buyer = seedSecurityUser(t, gdb, seedOpts{Role: entity.UserRoleBuyer})
	tok, _ = issueSecurityToken(buyer, true)

	rec = securityPUT(e, "/me/profile", tok, `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty update status=%d want 400", rec.Code)
	}

	rec = securityPUT(e, "/me/profile", tok, `{"avatar_url":"not-a-url"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad avatar status=%d want 400 body=%s", rec.Code, rec.Body.String())
	}
}

func securityPUT(e *echo.Echo, path, token, body string) *httptest.ResponseRecorder {
	var (
		rec *httptest.ResponseRecorder
		req *http.Request
	)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	e.ServeHTTP(rec, req)
	return rec
}
