//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"crowdfund-app/model"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func TestFavoritesUnauthorized(t *testing.T) {
	var (
		e   *echo.Echo
		rec *httptest.ResponseRecorder
	)
	e = setupApp(t)
	rec = httpGET(e, "/favorites", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", rec.Code)
	}
}

func TestFavoritesAddListCheckRemove(t *testing.T) {
	var (
		e       *echo.Echo
		gdb     *gorm.DB
		seller  *model.User
		buyer   *model.User
		project *model.Project
		tok     string
		rec     *httptest.ResponseRecorder
		body    favoriteStatusBody
		list    favoritesListBody
	)
	e = setupApp(t)
	gdb = testDB(t)
	seller = seedUser(t, gdb, seedOpts{Role: model.UserRoleSeller})
	buyer = seedUser(t, gdb, seedOpts{Role: model.UserRoleBuyer})
	project = seedProject(t, gdb, seller.ID, model.ProjectStatusActive)
	tok, _ = issueToken(buyer, true)

	rec = httpPOST(e, "/favorites", tok, fmt.Sprintf(`{"project_id":"%s","action":"add"}`, project.ID))
	if rec.Code != http.StatusOK {
		t.Fatalf("add status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.IsFavorited {
		t.Fatalf("is_favorited=%v want true", body.IsFavorited)
	}

	rec = httpPOST(e, "/favorites", tok, fmt.Sprintf(`{"project_id":"%s","action":"add"}`, project.ID))
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate add status=%d want 409", rec.Code)
	}

	rec = httpGET(e, "/favorites/check/"+project.ID.String(), tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("check status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httpGET(e, "/favorites", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Favorites) != 1 {
		t.Fatalf("favorites len=%d want 1", len(list.Favorites))
	}

	rec = httpPOST(e, "/favorites", tok, fmt.Sprintf(`{"project_id":"%s","action":"remove"}`, project.ID))
	if rec.Code != http.StatusOK {
		t.Fatalf("remove status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httpGET(e, "/favorites/check/"+project.ID.String(), tok)
	body = favoriteStatusBody{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.IsFavorited {
		t.Fatalf("is_favorited=%v want false", body.IsFavorited)
	}
}

func TestFavoritesPendingProjectNotFound(t *testing.T) {
	var (
		e       *echo.Echo
		gdb     *gorm.DB
		seller  *model.User
		buyer   *model.User
		project *model.Project
		tok     string
		rec     *httptest.ResponseRecorder
	)
	e = setupApp(t)
	gdb = testDB(t)
	seller = seedUser(t, gdb, seedOpts{Role: model.UserRoleSeller})
	buyer = seedUser(t, gdb, seedOpts{Role: model.UserRoleBuyer})
	project = seedProject(t, gdb, seller.ID, model.ProjectStatusPending)
	tok, _ = issueToken(buyer, true)

	rec = httpPOST(e, "/favorites", tok, fmt.Sprintf(`{"project_id":"%s","action":"add"}`, project.ID))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404 for pending project", rec.Code)
	}
}

func TestFavoritesRemoveMissing(t *testing.T) {
	var (
		e     *echo.Echo
		gdb   *gorm.DB
		buyer *model.User
		tok   string
		rec   *httptest.ResponseRecorder
		pid   uuid.UUID
	)
	e = setupApp(t)
	gdb = testDB(t)
	buyer = seedUser(t, gdb, seedOpts{Role: model.UserRoleBuyer})
	tok, _ = issueToken(buyer, true)
	pid = uuid.New()

	rec = httpPOST(e, "/favorites", tok, fmt.Sprintf(`{"project_id":"%s","action":"remove"}`, pid))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", rec.Code)
	}
}
