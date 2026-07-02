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
	"crowdfund-app/stripe"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type upgradeFixture struct {
	seller    *model.User
	buyer     *model.User
	project   *model.Project
	bronze    *model.ProjectPlan
	silver    *model.ProjectPlan
	supporter *model.ProjectSupporter
}

type upgradeResponse struct {
	Supporter struct {
		PlanID string `json:"plan_id"`
	} `json:"supporter"`
	PaymentIntentClientSecret *string `json:"payment_intent_client_secret"`
	PaymentIntentID           *string `json:"payment_intent_id"`
}

func seedUpgradeFixture(t *testing.T, gdb *gorm.DB) upgradeFixture {
	var (
		now       time.Time
		fx        upgradeFixture
		conn      *model.ConnectAccount
		supporter *model.ProjectSupporter
		err       error
	)
	t.Helper()
	now = time.Now().UTC()
	fx.seller = seedUser(t, gdb, seedOpts{Role: model.UserRoleSeller, TOTPEnabled: true})
	fx.buyer = seedUser(t, gdb, seedOpts{Role: model.UserRoleBuyer})
	fx.project = seedProject(t, gdb, fx.seller.ID, model.ProjectStatusActive)
	fx.bronze = &model.ProjectPlan{
		ID:             uuid.New(),
		ProjectID:      fx.project.ID,
		MembershipType: model.MembershipBronze,
		Price:          1000,
		Benefits:       "bronze",
		IsActive:       true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	fx.silver = &model.ProjectPlan{
		ID:             uuid.New(),
		ProjectID:      fx.project.ID,
		MembershipType: model.MembershipSilver,
		Price:          2000,
		Benefits:       "silver",
		IsActive:       true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err = gdb.Create(fx.bronze).Error; err != nil {
		t.Fatal(err)
	}
	if err = gdb.Create(fx.silver).Error; err != nil {
		t.Fatal(err)
	}
	conn = &model.ConnectAccount{
		ID:               uuid.New(),
		UserID:           fx.seller.ID,
		StripeAccountID:  fmt.Sprintf("acct_upgrade_%s", uuid.New()),
		ChargesEnabled:   true,
		PayoutsEnabled:   true,
		DetailsSubmitted: true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err = gdb.Create(conn).Error; err != nil {
		t.Fatal(err)
	}
	supporter = &model.ProjectSupporter{
		ID:                 uuid.New(),
		ProjectID:          fx.project.ID,
		UserID:             fx.buyer.ID,
		PlanID:             fx.bronze.ID,
		Status:             model.SupporterStatusActive,
		BillingMode:        model.BillingModeAuto,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 0, 30),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err = gdb.Create(supporter).Error; err != nil {
		t.Fatal(err)
	}
	fx.supporter = supporter
	return fx
}

func TestPlanUpgradePaidTier(t *testing.T) {
	var (
		e       *echo.Echo
		gdb     *gorm.DB
		fx      upgradeFixture
		tok     string
		rec     *httptest.ResponseRecorder
		payload upgradeResponse
		updated model.ProjectSupporter
		err     error
	)
	e, _ = setupPaymentApp(t)
	gdb = testDB(t)
	fx = seedUpgradeFixture(t, gdb)
	tok, _ = issueToken(fx.buyer, true)

	rec = httpPOST(e, fmt.Sprintf("/supporters/%s/upgrade", fx.supporter.ID), tok, fmt.Sprintf(
		`{"plan_id":%q}`,
		fx.silver.ID,
	))
	if rec.Code != http.StatusOK {
		t.Fatalf("upgrade status=%d body=%s", rec.Code, rec.Body.String())
	}
	err = json.Unmarshal(rec.Body.Bytes(), &payload)
	if err != nil {
		t.Fatal(err)
	}
	if payload.PaymentIntentClientSecret != nil && *payload.PaymentIntentClientSecret != "" {
		if payload.PaymentIntentID == nil || *payload.PaymentIntentID == "" {
			t.Fatal("missing payment_intent_id")
		}
		rec = httpPOST(e, fmt.Sprintf("/supporters/%s/upgrade-confirm", fx.supporter.ID), tok, fmt.Sprintf(
			`{"payment_intent_id":%q}`,
			*payload.PaymentIntentID,
		))
		if rec.Code != http.StatusOK {
			t.Fatalf("upgrade-confirm status=%d body=%s", rec.Code, rec.Body.String())
		}
	}

	err = gdb.Where("id = ?", fx.supporter.ID).First(&updated).Error
	if err != nil {
		t.Fatal(err)
	}
	if updated.PlanID != fx.silver.ID {
		t.Fatalf("plan_id=%s want %s", updated.PlanID, fx.silver.ID)
	}
	if updated.PendingPlanID != nil {
		t.Fatalf("pending plan still set: %v", updated.PendingPlanID)
	}
}

func TestPlanDowngradeSchedulesChange(t *testing.T) {
	var (
		e         *echo.Echo
		gdb       *gorm.DB
		fx        upgradeFixture
		tok       string
		rec       *httptest.ResponseRecorder
		goldPlan  *model.ProjectPlan
		updated   model.ProjectSupporter
		now       time.Time
		err       error
	)
	e, _ = setupPaymentApp(t)
	gdb = testDB(t)
	fx = seedUpgradeFixture(t, gdb)
	now = time.Now().UTC()
	goldPlan = &model.ProjectPlan{
		ID:             uuid.New(),
		ProjectID:      fx.project.ID,
		MembershipType: model.MembershipGold,
		Price:          3000,
		Benefits:       "gold",
		IsActive:       true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err = gdb.Create(goldPlan).Error; err != nil {
		t.Fatal(err)
	}
	err = gdb.Model(fx.supporter).Updates(map[string]interface{}{
		"plan_id": goldPlan.ID,
	}).Error
	if err != nil {
		t.Fatal(err)
	}
	tok, _ = issueToken(fx.buyer, true)

	rec = httpPOST(e, fmt.Sprintf("/supporters/%s/upgrade", fx.supporter.ID), tok, fmt.Sprintf(
		`{"plan_id":%q}`,
		fx.bronze.ID,
	))
	if rec.Code != http.StatusOK {
		t.Fatalf("downgrade status=%d body=%s", rec.Code, rec.Body.String())
	}

	err = gdb.Where("id = ?", fx.supporter.ID).First(&updated).Error
	if err != nil {
		t.Fatal(err)
	}
	if updated.PendingPlanID == nil || *updated.PendingPlanID != fx.bronze.ID {
		t.Fatalf("pending_plan_id=%v want %s", updated.PendingPlanID, fx.bronze.ID)
	}
	if updated.PlanID != goldPlan.ID {
		t.Fatalf("current plan should remain gold until period end")
	}
}

func TestPlanUpgradeVerifyIntentPath(t *testing.T) {
	var (
		e        *echo.Echo
		stub     *stripe.StubClient
		gdb      *gorm.DB
		fx       upgradeFixture
		tok      string
		rec      *httptest.ResponseRecorder
		intentID string
		payload  upgradeResponse
		updated  model.ProjectSupporter
		err      error
	)
	e, stub = setupPaymentApp(t)
	gdb = testDB(t)
	fx = seedUpgradeFixture(t, gdb)
	tok, _ = issueToken(fx.buyer, true)

	rec = httpPOST(e, fmt.Sprintf("/supporters/%s/upgrade", fx.supporter.ID), tok, fmt.Sprintf(
		`{"plan_id":%q}`,
		fx.silver.ID,
	))
	if rec.Code != http.StatusOK {
		t.Fatalf("upgrade status=%d body=%s", rec.Code, rec.Body.String())
	}
	err = json.Unmarshal(rec.Body.Bytes(), &payload)
	if err != nil {
		t.Fatal(err)
	}
	if payload.PaymentIntentID == nil || *payload.PaymentIntentID == "" {
		t.Skip("stub finalized upgrade without payment intent")
	}
	intentID = *payload.PaymentIntentID
	stub.SetIntentSucceeded(intentID)

	rec = httpPOST(e, "/payments/verify/intent", tok, fmt.Sprintf(`{"payment_intent_id":%q}`, intentID))
	if rec.Code != http.StatusOK {
		t.Fatalf("verify intent status=%d body=%s", rec.Code, rec.Body.String())
	}

	err = gdb.Where("id = ?", fx.supporter.ID).First(&updated).Error
	if err != nil {
		t.Fatal(err)
	}
	if updated.PlanID != fx.silver.ID {
		t.Fatalf("plan_id=%s want %s after verify", updated.PlanID, fx.silver.ID)
	}
}
