//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"crowdfund-app/model"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func TestSellerDashboardForbiddenForBuyer(t *testing.T) {
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
	rec = httpGET(e, "/seller/dashboard/stats", tok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rec.Code)
	}
}

func TestSellerDashboardStatsAndSupporters(t *testing.T) {
	var (
		e        *echo.Echo
		gdb      *gorm.DB
		fx       sellerFixture
		tok      string
		rec      *httptest.ResponseRecorder
		stats    sellerStatsBody
		supports sellerSupportersBody
		analysis sellerProjectsAnalysisBody
	)
	e = setupApp(t)
	gdb = testDB(t)
	fx = seedSellerFixture(t, gdb)
	tok, _ = issueToken(fx.seller, true)

	rec = httpGET(e, "/seller/dashboard/stats", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("stats status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &stats); err != nil {
		t.Fatal(err)
	}
	if stats.TotalProjects != 1 {
		t.Fatalf("total_projects=%d", stats.TotalProjects)
	}
	if stats.TotalSupporters != 1 {
		t.Fatalf("total_supporters=%d", stats.TotalSupporters)
	}
	if stats.TotalRevenue != 1000 {
		t.Fatalf("total_revenue=%d", stats.TotalRevenue)
	}

	rec = httpGET(e, fmt.Sprintf("/seller/supporters?project_id=%s", fx.project.ID), tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &supports); err != nil {
		t.Fatal(err)
	}
	if len(supports.Supporters) != 1 {
		t.Fatalf("supporters len=%d want 1", len(supports.Supporters))
	}

	rec = httpGET(e, "/seller/supporters/export", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("export status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/csv") {
		t.Fatalf("content-type=%q", rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(rec.Body.String(), fx.buyer.ID.String()) {
		t.Fatalf("csv missing buyer id: %s", rec.Body.String())
	}

	rec = httpGET(e, "/seller/projects/analysis", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("analysis status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &analysis); err != nil {
		t.Fatal(err)
	}
	if len(analysis.Projects) != 1 {
		t.Fatalf("projects len=%d want 1", len(analysis.Projects))
	}
}

type sellerFixture struct {
	seller  *model.User
	buyer   *model.User
	project *model.Project
	plan    *model.ProjectPlan
}

func seedSellerFixture(t *testing.T, gdb *gorm.DB) sellerFixture {
	var (
		now     time.Time
		seller  *model.User
		buyer   *model.User
		project *model.Project
		plan    *model.ProjectPlan
		err     error
	)
	t.Helper()
	now = time.Now().UTC()
	seller = seedUser(t, gdb, seedOpts{Role: model.UserRoleSeller, TOTPEnabled: true})
	buyer = seedUser(t, gdb, seedOpts{Role: model.UserRoleBuyer})
	project = seedProject(t, gdb, seller.ID, model.ProjectStatusActive)
	err = gdb.Model(project).Update("supporter_count", 1).Error
	if err != nil {
		t.Fatal(err)
	}
	plan = &model.ProjectPlan{
		ID:             uuid.New(),
		ProjectID:      project.ID,
		MembershipType: model.MembershipBronze,
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
	err = gdb.Create(&model.ProjectSupporter{
		ID:                 uuid.New(),
		ProjectID:          project.ID,
		UserID:             buyer.ID,
		PlanID:             plan.ID,
		Status:             model.SupporterStatusActive,
		BillingMode:        model.BillingModeKonbini,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 0, 30),
		CreatedAt:          now,
		UpdatedAt:          now,
	}).Error
	if err != nil {
		t.Fatal(err)
	}
	err = gdb.Create(&model.Purchase{
		ID:            uuid.New(),
		ProjectID:     project.ID,
		BuyerID:       buyer.ID,
		Quantity:      1,
		TotalPrice:    1000,
		Status:        model.PurchaseStatusPaid,
		PaymentMethod: model.PaymentMethodKonbini,
		CreatedAt:     now,
		UpdatedAt:     now,
	}).Error
	if err != nil {
		t.Fatal(err)
	}
	return sellerFixture{seller: seller, buyer: buyer, project: project, plan: plan}
}
