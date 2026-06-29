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

func TestReportsUnauthorized(t *testing.T) {
	var (
		e   *echo.Echo
		rec *httptest.ResponseRecorder
	)
	e = setupApp(t)
	rec = httpPOST(e, "/reports", "", `{"target_type":"PROJECT","target_id":"`+uuid.New().String()+`","reason":"spam"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", rec.Code)
	}
}

func TestReportsCreateListPatch(t *testing.T) {
	var (
		e       *echo.Echo
		gdb     *gorm.DB
		seller  *model.User
		buyer   *model.User
		admin   *model.User
		project *model.Project
		buyTok  string
		admTok  string
		rec     *httptest.ResponseRecorder
		body    reportBody
		id      string
		role    model.AdminRole
	)
	e = setupApp(t)
	gdb = openDB(t)
	seller = seedUser(t, gdb, seedOpts{Role: model.UserRoleSeller})
	buyer = seedUser(t, gdb, seedOpts{Role: model.UserRoleBuyer})
	project = seedProject(t, gdb, seller.ID, model.ProjectStatusActive)
	buyTok, _ = issueToken(buyer, true)
	role = model.AdminRoleSupportAdmin
	admin = seedUser(t, gdb, seedOpts{
		Role:        model.UserRoleAdmin,
		AdminRole:   &role,
		TOTPEnabled: true,
	})
	admTok, _ = issueToken(admin, true)

	rec = httpPOST(e, "/reports", buyTok, fmt.Sprintf(
		`{"target_type":"PROJECT","target_id":"%s","reason":"inappropriate content"}`,
		project.ID,
	))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	id = body.ID
	if id == "" {
		t.Fatal("missing report id")
	}

	rec = httpPOST(e, "/reports", buyTok, fmt.Sprintf(
		`{"target_type":"PROJECT","target_id":"%s","reason":"duplicate"}`,
		project.ID,
	))
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate status=%d want 409", rec.Code)
	}

	rec = httpGET(e, "/admin/reports", admTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httpPATCH(e, "/admin/reports/"+id, admTok, `{"status":"RESOLVED","resolution":"reviewed"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch status=%d body=%s", rec.Code, rec.Body.String())
	}
	body = reportBody{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Status != "RESOLVED" {
		t.Fatalf("status=%q", body.Status)
	}
}

func TestReportsFinanceAdminForbidden(t *testing.T) {
	var (
		e    *echo.Echo
		gdb  *gorm.DB
		u    *model.User
		tok  string
		role model.AdminRole
		rec  *httptest.ResponseRecorder
	)
	e = setupApp(t)
	gdb = openDB(t)
	role = model.AdminRoleFinanceAdmin
	u = seedUser(t, gdb, seedOpts{
		Role:        model.UserRoleAdmin,
		AdminRole:   &role,
		TOTPEnabled: true,
	})
	tok, _ = issueToken(u, true)
	rec = httpGET(e, "/admin/reports", tok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rec.Code)
	}
}

func TestReportsSelfUserTargetBadRequest(t *testing.T) {
	var (
		e      *echo.Echo
		gdb    *gorm.DB
		buyer  *model.User
		buyTok string
		rec    *httptest.ResponseRecorder
	)
	e = setupApp(t)
	gdb = openDB(t)
	buyer = seedUser(t, gdb, seedOpts{Role: model.UserRoleBuyer})
	buyTok, _ = issueToken(buyer, true)
	rec = httpPOST(e, "/reports", buyTok, fmt.Sprintf(
		`{"target_type":"USER","target_id":"%s","reason":"self report"}`,
		buyer.ID,
	))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400", rec.Code)
	}
}
