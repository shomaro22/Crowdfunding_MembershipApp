//go:build integration

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func TestSellerDashboardForbiddenForBuyer(t *testing.T) {
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
	rec = securityGET(e, "/seller/dashboard/stats", tok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rec.Code)
	}
}

func TestSellerDashboardStatsAndSupporters(t *testing.T) {
	var (
		e         *echo.Echo
		gdb       *gorm.DB
		fx        sellerFixture
		tok       string
		rec       *httptest.ResponseRecorder
		body      map[string]any
	)
	e = setupSecurityApp(t)
	gdb = openSecurityDB(t)
	fx = seedSellerFixture(t, gdb)
	tok, _ = issueSecurityToken(fx.seller, true)

	rec = securityGET(e, "/seller/dashboard/stats", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("stats status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if int64(body["total_projects"].(float64)) != 1 {
		t.Fatalf("total_projects=%v", body["total_projects"])
	}
	if int64(body["total_supporters"].(float64)) != 1 {
		t.Fatalf("total_supporters=%v", body["total_supporters"])
	}
	if int64(body["total_revenue"].(float64)) != 1000 {
		t.Fatalf("total_revenue=%v", body["total_revenue"])
	}

	rec = securityGET(e, fmt.Sprintf("/seller/supporters?project_id=%s", fx.project.ID), tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", rec.Code, rec.Body.String())
	}
	body = nil
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	supporters, _ := body["supporters"].([]any)
	if len(supporters) != 1 {
		t.Fatalf("supporters len=%d want 1", len(supporters))
	}

	rec = securityGET(e, "/seller/supporters/export", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("export status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/csv") {
		t.Fatalf("content-type=%q", rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(rec.Body.String(), fx.buyer.ID.String()) {
		t.Fatalf("csv missing buyer id: %s", rec.Body.String())
	}

	rec = securityGET(e, "/seller/projects/analysis", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("analysis status=%d body=%s", rec.Code, rec.Body.String())
	}
	body = nil
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	projects, _ := body["projects"].([]any)
	if len(projects) != 1 {
		t.Fatalf("projects len=%d want 1", len(projects))
	}
}

type sellerFixture struct {
	seller  *entity.User
	buyer   *entity.User
	project *entity.Project
	plan    *entity.ProjectPlan
}

func seedSellerFixture(t *testing.T, gdb *gorm.DB) sellerFixture {
	var (
		now    time.Time
		seller *entity.User
		buyer  *entity.User
		project *entity.Project
		plan   *entity.ProjectPlan
		err    error
	)
	t.Helper()
	now = time.Now().UTC()
	seller = seedSecurityUser(t, gdb, seedOpts{Role: entity.UserRoleSeller, TOTPEnabled: true})
	buyer = seedSecurityUser(t, gdb, seedOpts{Role: entity.UserRoleBuyer})
	project = seedSecurityProject(t, gdb, seller.ID, entity.ProjectStatusActive)
	err = gdb.Model(project).Updates(map[string]any{"supporter_count": 1}).Error
	if err != nil {
		t.Fatal(err)
	}
	plan = &entity.ProjectPlan{
		ID:             uuid.New(),
		ProjectID:      project.ID,
		MembershipType: entity.MembershipBronze,
		Price:          1000,
		Benefits:       "benefits",
		IsActive:       true,
		SupporterCount: 1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err = gdb.Create(plan).Error; err != nil {
		t.Fatal(err)
	}
	err = gdb.Create(&entity.ProjectSupporter{
		ID:                 uuid.New(),
		ProjectID:          project.ID,
		UserID:             buyer.ID,
		PlanID:             plan.ID,
		Status:             entity.SupporterStatusActive,
		BillingMode:        entity.BillingModeKonbini,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 0, 30),
		CreatedAt:          now,
		UpdatedAt:          now,
	}).Error
	if err != nil {
		t.Fatal(err)
	}
	err = gdb.Create(&entity.Purchase{
		ID:            uuid.New(),
		ProjectID:     project.ID,
		BuyerID:       buyer.ID,
		Quantity:      1,
		TotalPrice:    1000,
		Status:        entity.PurchaseStatusPaid,
		PaymentMethod: entity.PaymentMethodKonbini,
		CreatedAt:     now,
		UpdatedAt:     now,
	}).Error
	if err != nil {
		t.Fatal(err)
	}
	return sellerFixture{seller: seller, buyer: buyer, project: project, plan: plan}
}
