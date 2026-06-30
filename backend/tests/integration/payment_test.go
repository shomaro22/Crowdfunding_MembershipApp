//go:build integration

package integration

import (
	"bytes"
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

// §5.3 決済統合テスト。DATABASE_URL 必須。Stripe は StubClient を注入する。

type paymentFixture struct {
	seller  *model.User
	buyer   *model.User
	project *model.Project
	plan    *model.ProjectPlan
}

func seedPaymentFixture(t *testing.T, gdb *gorm.DB) paymentFixture {
	var (
		now    time.Time
		seller *model.User
		buyer  *model.User
		project   *model.Project
		plan   *model.ProjectPlan
		conn   *model.ConnectAccount
		err    error
	)
	t.Helper()
	now = time.Now().UTC()
	seller = seedUser(t, gdb, seedOpts{
		Role:        model.UserRoleSeller,
		TOTPEnabled: true,
	})
	buyer = seedUser(t, gdb, seedOpts{Role: model.UserRoleBuyer})
	project = seedProject(t, gdb, seller.ID, model.ProjectStatusActive)
	plan = &model.ProjectPlan{
		ID:             uuid.New(),
		ProjectID:      project.ID,
		MembershipType: model.MembershipBronze,
		Price:          1000,
		Benefits:       "test benefits",
		IsActive:       true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err = gdb.Create(plan).Error; err != nil {
		t.Fatal(err)
	}
	conn = &model.ConnectAccount{
		ID:               uuid.New(),
		UserID:           seller.ID,
		StripeAccountID:  fmt.Sprintf("acct_test_%s", uuid.New()),
		ChargesEnabled:   true,
		PayoutsEnabled:   true,
		DetailsSubmitted: true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err = gdb.Create(conn).Error; err != nil {
		t.Fatal(err)
	}
	return paymentFixture{seller: seller, buyer: buyer, project: project, plan: plan}
}

func TestPaymentKonbiniCheckoutAsyncWebhook(t *testing.T) {
	var (
		e         *echo.Echo
		stub      *stripe.StubClient
		gdb       *gorm.DB
		fx        paymentFixture
		tok       string
		rec       *httptest.ResponseRecorder
		sessionID string
		meta      map[string]string
		purchase  model.Purchase
		supporter model.ProjectSupporter
		project      model.Project
		err       error
	)
	e, stub = setupPaymentApp(t)
	gdb = testDB(t)
	fx = seedPaymentFixture(t, gdb)
	tok, _ = issueToken(fx.buyer, true)
	rec = httpPOST(e, "/payments/checkout", tok, fmt.Sprintf(
		`{"project_id":%q,"plan_id":%q,"payment_method":"KONBINI"}`,
		fx.project.ID, fx.plan.ID,
	))
	if rec.Code != http.StatusOK {
		t.Fatalf("checkout status=%d body=%s", rec.Code, rec.Body.String())
	}
	sessionID = jsonStringField(t, rec, "session_id")
	meta = stub.SessionMetadata(sessionID)
	rec = postStripeWebhook(t, e, "checkout.session.async_payment_succeeded", sessionID, meta, true)
	if rec.Code != http.StatusOK && rec.Code != http.StatusNoContent {
		t.Fatalf("webhook status=%d body=%s", rec.Code, rec.Body.String())
	}
	err = gdb.Where("stripe_checkout_session_id = ?", sessionID).First(&purchase).Error
	if err != nil {
		t.Fatal(err)
	}
	if purchase.Status != model.PurchaseStatusPaid {
		t.Fatalf("purchase status=%s want PAID", purchase.Status)
	}
	err = gdb.Where("project_id = ? AND user_id = ?", fx.project.ID, fx.buyer.ID).First(&supporter).Error
	if err != nil {
		t.Fatal(err)
	}
	if supporter.Status != model.SupporterStatusActive {
		t.Fatalf("supporter status=%s want ACTIVE", supporter.Status)
	}
	if supporter.BillingMode != model.BillingModeKonbini {
		t.Fatalf("billing mode=%s want KONBINI", supporter.BillingMode)
	}
	err = gdb.Where("id = ?", fx.project.ID).First(&project).Error
	if err != nil {
		t.Fatal(err)
	}
	if project.CurrentAmount != int64(fx.plan.Price) {
		t.Fatalf("current_amount=%d want %d", project.CurrentAmount, fx.plan.Price)
	}
}

func TestPaymentAutoIntentWebhook(t *testing.T) {
	var (
		e         *echo.Echo
		stub      *stripe.StubClient
		gdb       *gorm.DB
		fx        paymentFixture
		tok       string
		rec       *httptest.ResponseRecorder
		intentID  string
		meta      map[string]string
		purchase  model.Purchase
		supporter model.ProjectSupporter
		err       error
	)
	e, stub = setupPaymentApp(t)
	gdb = testDB(t)
	fx = seedPaymentFixture(t, gdb)
	tok, _ = issueToken(fx.buyer, true)
	rec = httpPOST(e, "/payments/intent", tok, fmt.Sprintf(
		`{"project_id":%q,"plan_id":%q}`,
		fx.project.ID, fx.plan.ID,
	))
	if rec.Code != http.StatusOK {
		t.Fatalf("intent status=%d body=%s", rec.Code, rec.Body.String())
	}
	intentID = jsonStringField(t, rec, "payment_intent_id")
	meta = stub.IntentMetadata(intentID)
	rec = postStripeWebhook(t, e, "payment_intent.succeeded", intentID, meta, false)
	if rec.Code != http.StatusOK && rec.Code != http.StatusNoContent {
		t.Fatalf("webhook status=%d body=%s", rec.Code, rec.Body.String())
	}
	err = gdb.Where("stripe_payment_intent_id = ?", intentID).First(&purchase).Error
	if err != nil {
		t.Fatal(err)
	}
	if purchase.Status != model.PurchaseStatusPaid {
		t.Fatalf("purchase status=%s want PAID", purchase.Status)
	}
	err = gdb.Where("project_id = ? AND user_id = ?", fx.project.ID, fx.buyer.ID).First(&supporter).Error
	if err != nil {
		t.Fatal(err)
	}
	if supporter.BillingMode != model.BillingModeAuto {
		t.Fatalf("billing mode=%s want AUTO", supporter.BillingMode)
	}
}

func TestPaymentVerifyCheckoutFallback(t *testing.T) {
	var (
		e         *echo.Echo
		stub      *stripe.StubClient
		gdb       *gorm.DB
		fx        paymentFixture
		tok       string
		rec       *httptest.ResponseRecorder
		sessionID string
		purchase  model.Purchase
		err       error
	)
	e, stub = setupPaymentApp(t)
	gdb = testDB(t)
	fx = seedPaymentFixture(t, gdb)
	tok, _ = issueToken(fx.buyer, true)
	rec = httpPOST(e, "/payments/checkout", tok, fmt.Sprintf(
		`{"project_id":%q,"plan_id":%q,"payment_method":"KONBINI"}`,
		fx.project.ID, fx.plan.ID,
	))
	if rec.Code != http.StatusOK {
		t.Fatalf("checkout status=%d", rec.Code)
	}
	sessionID = jsonStringField(t, rec, "session_id")
	stub.SetSessionPaid(sessionID)
	rec = httpPOST(e, "/payments/verify/checkout", tok, fmt.Sprintf(`{"session_id":%q}`, sessionID))
	if rec.Code != http.StatusOK {
		t.Fatalf("verify status=%d body=%s", rec.Code, rec.Body.String())
	}
	err = gdb.Where("stripe_checkout_session_id = ?", sessionID).First(&purchase).Error
	if err != nil {
		t.Fatal(err)
	}
	if purchase.Status != model.PurchaseStatusPaid {
		t.Fatalf("purchase status=%s want PAID", purchase.Status)
	}
}

func TestPaymentWebhookIdempotent(t *testing.T) {
	var (
		e         *echo.Echo
		stub      *stripe.StubClient
		gdb       *gorm.DB
		fx        paymentFixture
		tok       string
		rec       *httptest.ResponseRecorder
		sessionID string
		meta      map[string]string
		eventID   string
		count     int64
		err       error
	)
	e, stub = setupPaymentApp(t)
	gdb = testDB(t)
	fx = seedPaymentFixture(t, gdb)
	tok, _ = issueToken(fx.buyer, true)
	rec = httpPOST(e, "/payments/checkout", tok, fmt.Sprintf(
		`{"project_id":%q,"plan_id":%q,"payment_method":"KONBINI"}`,
		fx.project.ID, fx.plan.ID,
	))
	if rec.Code != http.StatusOK {
		t.Fatal(rec.Body.String())
	}
	sessionID = jsonStringField(t, rec, "session_id")
	meta = stub.SessionMetadata(sessionID)
	eventID = fmt.Sprintf("evt_idempotent_%s", uuid.New())
	rec = postStripeWebhookWithID(t, e, eventID, "checkout.session.async_payment_succeeded", sessionID, meta, true)
	if rec.Code != http.StatusOK && rec.Code != http.StatusNoContent {
		t.Fatalf("first webhook status=%d", rec.Code)
	}
	rec = postStripeWebhookWithID(t, e, eventID, "checkout.session.async_payment_succeeded", sessionID, meta, true)
	if rec.Code != http.StatusOK && rec.Code != http.StatusNoContent {
		t.Fatalf("duplicate webhook status=%d", rec.Code)
	}
	err = gdb.Model(&model.ProjectSupporter{}).
		Where("project_id = ? AND user_id = ?", fx.project.ID, fx.buyer.ID).
		Count(&count).Error
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("supporter count=%d want 1", count)
	}
}

func TestPaymentKonbiniCheckoutExpired(t *testing.T) {
	var (
		e         *echo.Echo
		gdb       *gorm.DB
		fx        paymentFixture
		tok       string
		rec       *httptest.ResponseRecorder
		sessionID string
		meta      map[string]string
		purchase  model.Purchase
		err       error
	)
	e, _ = setupPaymentApp(t)
	gdb = testDB(t)
	fx = seedPaymentFixture(t, gdb)
	tok, _ = issueToken(fx.buyer, true)
	rec = httpPOST(e, "/payments/checkout", tok, fmt.Sprintf(
		`{"project_id":%q,"plan_id":%q,"payment_method":"KONBINI"}`,
		fx.project.ID, fx.plan.ID,
	))
	if rec.Code != http.StatusOK {
		t.Fatal(rec.Body.String())
	}
	sessionID = jsonStringField(t, rec, "session_id")
	meta = map[string]string{
		stripe.MetaProjectID: fx.project.ID.String(),
		stripe.MetaPlanID:    fx.plan.ID.String(),
	}
	rec = postStripeWebhook(t, e, "checkout.session.expired", sessionID, meta, true)
	if rec.Code != http.StatusOK && rec.Code != http.StatusNoContent {
		t.Fatalf("webhook status=%d body=%s", rec.Code, rec.Body.String())
	}
	err = gdb.Where("stripe_checkout_session_id = ?", sessionID).First(&purchase).Error
	if err != nil {
		t.Fatal(err)
	}
	if purchase.Status != model.PurchaseStatusCancelled {
		t.Fatalf("purchase status=%s want CANCELLED", purchase.Status)
	}
}

func TestPaymentAlreadySupportingConflict(t *testing.T) {
	var (
		e    *echo.Echo
		gdb  *gorm.DB
		fx   paymentFixture
		tok  string
		rec  *httptest.ResponseRecorder
		now  time.Time
		err  error
	)
	e, _ = setupPaymentApp(t)
	gdb = testDB(t)
	fx = seedPaymentFixture(t, gdb)
	now = time.Now().UTC()
	err = gdb.Create(&model.ProjectSupporter{
		ID:                 uuid.New(),
		ProjectID:          fx.project.ID,
		UserID:             fx.buyer.ID,
		PlanID:             fx.plan.ID,
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
	tok, _ = issueToken(fx.buyer, true)
	rec = httpPOST(e, "/payments/intent", tok, fmt.Sprintf(
		`{"project_id":%q,"plan_id":%q}`,
		fx.project.ID, fx.plan.ID,
	))
	if rec.Code != http.StatusConflict {
		t.Fatalf("status=%d want 409 body=%s", rec.Code, rec.Body.String())
	}
	if bodyCode(rec) != "CONFLICT" {
		t.Fatalf("code=%q want CONFLICT", bodyCode(rec))
	}
}

func postStripeWebhook(t *testing.T, e *echo.Echo, eventType, sessionOrIntentID string, meta map[string]string, konbini bool) *httptest.ResponseRecorder {
	t.Helper()
	return postStripeWebhookWithID(t, e, fmt.Sprintf("evt_%s", uuid.New()), eventType, sessionOrIntentID, meta, konbini)
}

func postStripeWebhookWithID(t *testing.T, e *echo.Echo, eventID, eventType, sessionOrIntentID string, meta map[string]string, konbini bool) *httptest.ResponseRecorder {
	var (
		payload []byte
		rec     *httptest.ResponseRecorder
		req     *http.Request
		err     error
	)
	t.Helper()
	payload, err = buildStripeEventPayload(eventID, eventType, sessionOrIntentID, meta, konbini)
	if err != nil {
		t.Fatal(err)
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/stripe/webhook", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", stripe.StubWebhookSignature)
	e.ServeHTTP(rec, req)
	return rec
}
