//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"crowdfund-app/model"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func TestProfileUnauthorized(t *testing.T) {
	var (
		e   *echo.Echo
		rec *httptest.ResponseRecorder
	)
	e = setupApp(t)
	rec = httpGET(e, "/me/profile", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("get status=%d want 401", rec.Code)
	}
	rec = httpPUT(e, "/me/profile", "", `{"name":"Alice"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("put status=%d want 401", rec.Code)
	}
}

func TestProfileGetAndUpdate(t *testing.T) {
	var (
		e      *echo.Echo
		gdb    *gorm.DB
		buyer  *model.User
		tok    string
		rec    *httptest.ResponseRecorder
		body   userProfileBody
	)
	e = setupApp(t)
	gdb = testDB(t)
	buyer = seedUser(t, gdb, seedOpts{Role: model.UserRoleBuyer})
	tok, _ = issueToken(buyer, true)

	rec = httpGET(e, "/me/profile", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Email != buyer.Email {
		t.Fatalf("email=%q want %s", body.Email, buyer.Email)
	}

	rec = httpPUT(e, "/me/profile", tok, `{"name":"Alice","avatar_url":"https://example.com/a.png"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("put status=%d body=%s", rec.Code, rec.Body.String())
	}
	body = userProfileBody{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Name != "Alice" {
		t.Fatalf("name=%q want Alice", body.Name)
	}
	if body.AvatarURL != "https://example.com/a.png" {
		t.Fatalf("avatar_url=%q", body.AvatarURL)
	}

	rec = httpGET(e, "/me/profile", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("get after update status=%d body=%s", rec.Code, rec.Body.String())
	}
	body = userProfileBody{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Name != "Alice" {
		t.Fatalf("persisted name=%q want Alice", body.Name)
	}
}

func TestProfileUpdateBadRequest(t *testing.T) {
	var (
		e     *echo.Echo
		gdb   *gorm.DB
		buyer *model.User
		tok   string
		rec   *httptest.ResponseRecorder
	)
	e = setupApp(t)
	gdb = testDB(t)
	buyer = seedUser(t, gdb, seedOpts{Role: model.UserRoleBuyer})
	tok, _ = issueToken(buyer, true)

	rec = httpPUT(e, "/me/profile", tok, `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty update status=%d want 400", rec.Code)
	}

	rec = httpPUT(e, "/me/profile", tok, `{"avatar_url":"not-a-url"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad avatar status=%d want 400 body=%s", rec.Code, rec.Body.String())
	}
}
