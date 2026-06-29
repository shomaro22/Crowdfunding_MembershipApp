//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"crowdfund-app/model"

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
	tok = issueAdminToken(t, gdb, model.AdminRoleFinanceAdmin)

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
		buyer *model.User
		rec   *httptest.ResponseRecorder
		body  userProfileBody
		name  = "Updated Name"
	)
	e = setupApp(t)
	gdb = openDB(t)
	buyer = seedUser(t, gdb, seedOpts{Role: model.UserRoleBuyer})
	tok = issueAdminToken(t, gdb, model.AdminRoleSupportAdmin)

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
	body = userProfileBody{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Name != name {
		t.Fatalf("name=%q want %q", body.Name, name)
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
		seller   *model.User
		buyer    *model.User
		project  *model.Project
		purchase *model.Purchase
		rec      *httptest.ResponseRecorder
		body     purchaseStatusBody
		now      time.Time
	)
	e = setupApp(t)
	gdb = openDB(t)
	now = time.Now().UTC()
	tok = issueAdminToken(t, gdb, model.AdminRoleSuperAdmin)
	seller = seedUser(t, gdb, seedOpts{Role: model.UserRoleSeller})
	buyer = seedUser(t, gdb, seedOpts{Role: model.UserRoleBuyer})
	project = seedProject(t, gdb, seller.ID, model.ProjectStatusActive)
	purchase = &model.Purchase{
		ID:            uuid.New(),
		ProjectID:     project.ID,
		BuyerID:       buyer.ID,
		Quantity:      1,
		TotalPrice:    500,
		Status:        model.PurchaseStatusPending,
		PaymentMethod: model.PaymentMethodCard,
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
	if body.Status != "CANCELLED" {
		t.Fatalf("status=%q want CANCELLED", body.Status)
	}
}
