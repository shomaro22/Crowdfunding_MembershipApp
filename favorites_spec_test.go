//go:build integration

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func TestFavoritesUnauthorized(t *testing.T) {
	var (
		e   *echo.Echo
		rec *httptest.ResponseRecorder
	)
	e = setupSecurityApp(t)
	rec = securityGET(e, "/favorites", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", rec.Code)
	}
}

func TestFavoritesAddListCheckRemove(t *testing.T) {
	var (
		e       *echo.Echo
		gdb     *gorm.DB
		seller  *entity.User
		buyer   *entity.User
		project *entity.Project
		tok     string
		rec     *httptest.ResponseRecorder
		body    map[string]any
	)
	e = setupSecurityApp(t)
	gdb = openSecurityDB(t)
	seller = seedSecurityUser(t, gdb, seedOpts{Role: entity.UserRoleSeller})
	buyer = seedSecurityUser(t, gdb, seedOpts{Role: entity.UserRoleBuyer})
	project = seedSecurityProject(t, gdb, seller.ID, entity.ProjectStatusActive)
	tok, _ = issueSecurityToken(buyer, true)

	rec = securityPOST(e, "/favorites", tok, fmt.Sprintf(`{"project_id":"%s","action":"add"}`, project.ID))
	if rec.Code != http.StatusOK {
		t.Fatalf("add status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["is_favorited"] != true {
		t.Fatalf("is_favorited=%v want true", body["is_favorited"])
	}

	rec = securityPOST(e, "/favorites", tok, fmt.Sprintf(`{"project_id":"%s","action":"add"}`, project.ID))
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate add status=%d want 409", rec.Code)
	}

	rec = securityGET(e, "/favorites/check/"+project.ID.String(), tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("check status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = securityGET(e, "/favorites", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", rec.Code, rec.Body.String())
	}
	body = nil
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	favs, _ := body["favorites"].([]any)
	if len(favs) != 1 {
		t.Fatalf("favorites len=%d want 1", len(favs))
	}

	rec = securityPOST(e, "/favorites", tok, fmt.Sprintf(`{"project_id":"%s","action":"remove"}`, project.ID))
	if rec.Code != http.StatusOK {
		t.Fatalf("remove status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = securityGET(e, "/favorites/check/"+project.ID.String(), tok)
	body = nil
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["is_favorited"] != false {
		t.Fatalf("is_favorited=%v want false", body["is_favorited"])
	}
}

func TestFavoritesPendingProjectNotFound(t *testing.T) {
	var (
		e       *echo.Echo
		gdb     *gorm.DB
		seller  *entity.User
		buyer   *entity.User
		project *entity.Project
		tok     string
		rec     *httptest.ResponseRecorder
	)
	e = setupSecurityApp(t)
	gdb = openSecurityDB(t)
	seller = seedSecurityUser(t, gdb, seedOpts{Role: entity.UserRoleSeller})
	buyer = seedSecurityUser(t, gdb, seedOpts{Role: entity.UserRoleBuyer})
	project = seedSecurityProject(t, gdb, seller.ID, entity.ProjectStatusPending)
	tok, _ = issueSecurityToken(buyer, true)

	rec = securityPOST(e, "/favorites", tok, fmt.Sprintf(`{"project_id":"%s","action":"add"}`, project.ID))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404 for pending project", rec.Code)
	}
}

func TestFavoritesRemoveMissing(t *testing.T) {
	var (
		e      *echo.Echo
		gdb    *gorm.DB
		buyer  *entity.User
		tok    string
		rec    *httptest.ResponseRecorder
		pid    uuid.UUID
	)
	e = setupSecurityApp(t)
	gdb = openSecurityDB(t)
	buyer = seedSecurityUser(t, gdb, seedOpts{Role: entity.UserRoleBuyer})
	tok, _ = issueSecurityToken(buyer, true)
	pid = uuid.New()

	rec = securityPOST(e, "/favorites", tok, fmt.Sprintf(`{"project_id":"%s","action":"remove"}`, pid))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", rec.Code)
	}
}
