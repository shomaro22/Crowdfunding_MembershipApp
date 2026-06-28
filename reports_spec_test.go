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

func TestReportsUnauthorized(t *testing.T) {
	var (
		e   *echo.Echo
		rec *httptest.ResponseRecorder
	)
	e = setupSecurityApp(t)
	rec = securityPOST(e, "/reports", "", `{"target_type":"PROJECT","target_id":"`+uuid.New().String()+`","reason":"spam"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", rec.Code)
	}
}

func TestReportsCreateListPatch(t *testing.T) {
	var (
		e       *echo.Echo
		gdb     *gorm.DB
		seller  *entity.User
		buyer   *entity.User
		admin   *entity.User
		project *entity.Project
		buyTok  string
		admTok  string
		rec     *httptest.ResponseRecorder
		body    map[string]any
		id      string
		role    entity.AdminRole
	)
	e = setupSecurityApp(t)
	gdb = openSecurityDB(t)
	seller = seedSecurityUser(t, gdb, seedOpts{Role: entity.UserRoleSeller})
	buyer = seedSecurityUser(t, gdb, seedOpts{Role: entity.UserRoleBuyer})
	project = seedSecurityProject(t, gdb, seller.ID, entity.ProjectStatusActive)
	buyTok, _ = issueSecurityToken(buyer, true)
	role = entity.AdminRoleSupportAdmin
	admin = seedSecurityUser(t, gdb, seedOpts{
		Role:        entity.UserRoleAdmin,
		AdminRole:   &role,
		TOTPEnabled: true,
	})
	admTok, _ = issueSecurityToken(admin, true)

	rec = securityPOST(e, "/reports", buyTok, fmt.Sprintf(
		`{"target_type":"PROJECT","target_id":"%s","reason":"inappropriate content"}`,
		project.ID,
	))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	id, _ = body["id"].(string)
	if id == "" {
		t.Fatal("missing report id")
	}

	rec = securityPOST(e, "/reports", buyTok, fmt.Sprintf(
		`{"target_type":"PROJECT","target_id":"%s","reason":"duplicate"}`,
		project.ID,
	))
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate status=%d want 409", rec.Code)
	}

	rec = securityGET(e, "/admin/reports", admTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = securityPATCH(e, "/admin/reports/"+id, admTok, `{"status":"RESOLVED","resolution":"reviewed"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch status=%d body=%s", rec.Code, rec.Body.String())
	}
	body = nil
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "RESOLVED" {
		t.Fatalf("status=%v", body["status"])
	}
}

func TestReportsFinanceAdminForbidden(t *testing.T) {
	var (
		e    *echo.Echo
		gdb  *gorm.DB
		u    *entity.User
		tok  string
		role entity.AdminRole
		rec  *httptest.ResponseRecorder
	)
	e = setupSecurityApp(t)
	gdb = openSecurityDB(t)
	role = entity.AdminRoleFinanceAdmin
	u = seedSecurityUser(t, gdb, seedOpts{
		Role:        entity.UserRoleAdmin,
		AdminRole:   &role,
		TOTPEnabled: true,
	})
	tok, _ = issueSecurityToken(u, true)
	rec = securityGET(e, "/admin/reports", tok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rec.Code)
	}
}

func TestReportsSelfUserTargetBadRequest(t *testing.T) {
	var (
		e      *echo.Echo
		gdb    *gorm.DB
		buyer  *entity.User
		buyTok string
		rec    *httptest.ResponseRecorder
	)
	e = setupSecurityApp(t)
	gdb = openSecurityDB(t)
	buyer = seedSecurityUser(t, gdb, seedOpts{Role: entity.UserRoleBuyer})
	buyTok, _ = issueSecurityToken(buyer, true)
	rec = securityPOST(e, "/reports", buyTok, fmt.Sprintf(
		`{"target_type":"USER","target_id":"%s","reason":"self report"}`,
		buyer.ID,
	))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400", rec.Code)
	}
}
