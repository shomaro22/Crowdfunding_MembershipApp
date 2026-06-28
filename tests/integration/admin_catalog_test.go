//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func TestAdminCatalogFinanceRBAC(t *testing.T) {
	var (
		e   *echo.Echo
		gdb *gorm.DB
		tok string
		rec *httptest.ResponseRecorder
	)
	e = setupApp(t)
	gdb = openDB(t)
	tok = issueAdminToken(t, gdb, entity.AdminRoleFinanceAdmin)

	rec = httpGET(e, "/admin/purchases", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("purchases status=%d want 200", rec.Code)
	}

	rec = httpGET(e, "/admin/stats", tok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("stats status=%d want 403", rec.Code)
	}
}

func TestAdminCatalogSupportUserManagement(t *testing.T) {
	var (
		e     *echo.Echo
		gdb   *gorm.DB
		tok   string
		buyer *entity.User
		rec   *httptest.ResponseRecorder
		body  map[string]any
		name  = "Updated Name"
	)
	e = setupApp(t)
	gdb = openDB(t)
	buyer = seedUser(t, gdb, seedOpts{Role: entity.UserRoleBuyer})
	tok = issueAdminToken(t, gdb, entity.AdminRoleSupportAdmin)

	rec = httpGET(e, "/admin/stats", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("stats status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httpGET(e, "/admin/users?role=BUYER", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("users status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httpPUT(e, fmt.Sprintf("/admin/users/%s", buyer.ID), tok, fmt.Sprintf(`{"name":%q}`, name))
	if rec.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", rec.Code, rec.Body.String())
	}
	body = nil
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["name"] != name {
		t.Fatalf("name=%v want %q", body["name"], name)
	}

	rec = httpGET(e, "/admin/purchases", tok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("purchases status=%d want 403", rec.Code)
	}
}

func TestAdminCatalogPatchPurchase(t *testing.T) {
	var (
		e        *echo.Echo
		gdb      *gorm.DB
		tok      string
		seller   *entity.User
		buyer    *entity.User
		project  *entity.Project
		purchase *entity.Purchase
		rec      *httptest.ResponseRecorder
		body     map[string]any
		now      time.Time
	)
	e = setupApp(t)
	gdb = openDB(t)
	now = time.Now().UTC()
	tok = issueAdminToken(t, gdb, entity.AdminRoleSuperAdmin)
	seller = seedUser(t, gdb, seedOpts{Role: entity.UserRoleSeller})
	buyer = seedUser(t, gdb, seedOpts{Role: entity.UserRoleBuyer})
	project = seedProject(t, gdb, seller.ID, entity.ProjectStatusActive)
	purchase = &entity.Purchase{
		ID:            uuid.New(),
		ProjectID:     project.ID,
		BuyerID:       buyer.ID,
		Quantity:      1,
		TotalPrice:    500,
		Status:        entity.PurchaseStatusPending,
		PaymentMethod: entity.PaymentMethodCard,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := gdb.Create(purchase).Error; err != nil {
		t.Fatal(err)
	}

	rec = httpPATCH(e, fmt.Sprintf("/admin/purchases/%s", purchase.ID), tok, `{"status":"CANCELLED"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "CANCELLED" {
		t.Fatalf("status=%v want CANCELLED", body["status"])
	}
}
