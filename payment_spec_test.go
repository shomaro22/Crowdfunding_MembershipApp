package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"crowdfund-app/db"
	"crowdfund-app/entity"
	"crowdfund-app/stripe"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// §5.3 決済統合テスト。DATABASE_URL 必須。Stripe は StubClient を注入する。

type paymentFixture struct {
	seller  *entity.User
	buyer   *entity.User
	project *entity.Project
	plan    *entity.ProjectPlan
}

func setupPaymentApp(t *testing.T) (*echo.Echo, *stripe.StubClient) {
	var (
		e     *echo.Echo
		gdb   *gorm.DB
		sqlDB *sql.DB
		stub  *stripe.StubClient
		err   error
	)
	t.Helper()
	if strings.TrimSpace(os.Getenv("DATABASE_URL")) == "" {
		t.Skip("DATABASE_URL が未設定のためスキップ")
	}
	gdb, err = dbOpenPaymentTest(t)
	if err != nil {
		t.Skip("DB 接続不可のためスキップ: ", err)
	}
	sqlDB, err = gdb.DB()
	if err != nil {
		t.Skip("DB 接続不可のためスキップ: ", err)
	}
	if err = sqlDB.Ping(); err != nil {
		t.Skip("DB 接続不可のためスキップ: ", err)
	}
	stub = stripe.NewStubClient()
	setTestStripeClient(stub)
	t.Cleanup(clearTestStripeClient)
	_ = os.Setenv("JWT_SECRET", securityTestJWTSecret)
	e, _, err = buildApp()
	if err != nil {
		t.Fatalf("buildApp: %v", err)
	}
	return e, stub
}

func dbOpenPaymentTest(t *testing.T) (*gorm.DB, error) {
	t.Helper()
	return db.Open(os.Getenv("DATABASE_URL"))
}

func seedPaymentFixture(t *testing.T, gdb *gorm.DB) paymentFixture {
	var (
		now    time.Time
		seller *entity.User
		buyer  *entity.User
		proj   *entity.Project
		plan   *entity.ProjectPlan
		conn   *entity.ConnectAccount
		err    error
	)
	t.Helper()
	now = time.Now().UTC()
	seller = seedSecurityUser(t, gdb, seedOpts{
		Role:        entity.UserRoleSeller,
		TOTPEnabled: true,
	})
	buyer = seedSecurityUser(t, gdb, seedOpts{Role: entity.UserRoleBuyer})
	proj = seedSecurityProject(t, gdb, seller.ID, entity.ProjectStatusActive)
	plan = &entity.ProjectPlan{
		ID:             uuid.New(),
		ProjectID:      proj.ID,
		MembershipType: entity.MembershipBronze,
		Price:          1000,
		Benefits:       "test benefits",
		IsActive:       true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err = gdb.Create(plan).Error; err != nil {
		t.Fatal(err)
	}
	conn = &entity.ConnectAccount{
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
	return paymentFixture{seller: seller, buyer: buyer, project: proj, plan: plan}
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
		purchase  entity.Purchase
		supporter entity.ProjectSupporter
		proj      entity.Project
		err       error
	)
	e, stub = setupPaymentApp(t)
	gdb = openSecurityDB(t)
	fx = seedPaymentFixture(t, gdb)
	tok, _ = issueSecurityToken(fx.buyer, true)
	rec = securityPOST(e, "/payments/checkout", tok, fmt.Sprintf(
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
	if purchase.Status != entity.PurchaseStatusPaid {
		t.Fatalf("purchase status=%s want PAID", purchase.Status)
	}
	err = gdb.Where("project_id = ? AND user_id = ?", fx.project.ID, fx.buyer.ID).First(&supporter).Error
	if err != nil {
		t.Fatal(err)
	}
	if supporter.Status != entity.SupporterStatusActive {
		t.Fatalf("supporter status=%s want ACTIVE", supporter.Status)
	}
	if supporter.BillingMode != entity.BillingModeKonbini {
		t.Fatalf("billing mode=%s want KONBINI", supporter.BillingMode)
	}
	err = gdb.Where("id = ?", fx.project.ID).First(&proj).Error
	if err != nil {
		t.Fatal(err)
	}
	if proj.CurrentAmount != int64(fx.plan.Price) {
		t.Fatalf("current_amount=%d want %d", proj.CurrentAmount, fx.plan.Price)
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
		purchase  entity.Purchase
		supporter entity.ProjectSupporter
		err       error
	)
	e, stub = setupPaymentApp(t)
	gdb = openSecurityDB(t)
	fx = seedPaymentFixture(t, gdb)
	tok, _ = issueSecurityToken(fx.buyer, true)
	rec = securityPOST(e, "/payments/intent", tok, fmt.Sprintf(
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
	if purchase.Status != entity.PurchaseStatusPaid {
		t.Fatalf("purchase status=%s want PAID", purchase.Status)
	}
	err = gdb.Where("project_id = ? AND user_id = ?", fx.project.ID, fx.buyer.ID).First(&supporter).Error
	if err != nil {
		t.Fatal(err)
	}
	if supporter.BillingMode != entity.BillingModeAuto {
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
		purchase  entity.Purchase
		err       error
	)
	e, stub = setupPaymentApp(t)
	gdb = openSecurityDB(t)
	fx = seedPaymentFixture(t, gdb)
	tok, _ = issueSecurityToken(fx.buyer, true)
	rec = securityPOST(e, "/payments/checkout", tok, fmt.Sprintf(
		`{"project_id":%q,"plan_id":%q,"payment_method":"KONBINI"}`,
		fx.project.ID, fx.plan.ID,
	))
	if rec.Code != http.StatusOK {
		t.Fatalf("checkout status=%d", rec.Code)
	}
	sessionID = jsonStringField(t, rec, "session_id")
	stub.SetSessionPaid(sessionID)
	rec = securityPOST(e, "/payments/verify/checkout", tok, fmt.Sprintf(`{"session_id":%q}`, sessionID))
	if rec.Code != http.StatusOK {
		t.Fatalf("verify status=%d body=%s", rec.Code, rec.Body.String())
	}
	err = gdb.Where("stripe_checkout_session_id = ?", sessionID).First(&purchase).Error
	if err != nil {
		t.Fatal(err)
	}
	if purchase.Status != entity.PurchaseStatusPaid {
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
	gdb = openSecurityDB(t)
	fx = seedPaymentFixture(t, gdb)
	tok, _ = issueSecurityToken(fx.buyer, true)
	rec = securityPOST(e, "/payments/checkout", tok, fmt.Sprintf(
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
	err = gdb.Model(&entity.ProjectSupporter{}).
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
		purchase  entity.Purchase
		err       error
	)
	e, _ = setupPaymentApp(t)
	gdb = openSecurityDB(t)
	fx = seedPaymentFixture(t, gdb)
	tok, _ = issueSecurityToken(fx.buyer, true)
	rec = securityPOST(e, "/payments/checkout", tok, fmt.Sprintf(
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
	if purchase.Status != entity.PurchaseStatusCancelled {
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
	gdb = openSecurityDB(t)
	fx = seedPaymentFixture(t, gdb)
	now = time.Now().UTC()
	err = gdb.Create(&entity.ProjectSupporter{
		ID:                 uuid.New(),
		ProjectID:          fx.project.ID,
		UserID:             fx.buyer.ID,
		PlanID:             fx.plan.ID,
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
	tok, _ = issueSecurityToken(fx.buyer, true)
	rec = securityPOST(e, "/payments/intent", tok, fmt.Sprintf(
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

func buildStripeEventPayload(eventID, eventType, objectID string, meta map[string]string, konbini bool) ([]byte, error) {
	var (
		obj     map[string]any
		objRaw  []byte
		payload []byte
		err     error
	)
	obj = map[string]any{
		"id":       objectID,
		"metadata": meta,
	}
	switch eventType {
	case "payment_intent.succeeded":
		obj["status"] = "succeeded"
	case "checkout.session.async_payment_succeeded", "checkout.session.completed":
		obj["payment_status"] = "paid"
		if konbini {
			obj["payment_method_types"] = []string{"konbini"}
		}
	case "checkout.session.expired":
		if konbini {
			obj["payment_method_types"] = []string{"konbini"}
		}
	}
	objRaw, err = json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	payload, err = json.Marshal(map[string]any{
		"id":   eventID,
		"type": eventType,
		"data": map[string]any{
			"object": json.RawMessage(objRaw),
		},
	})
	return payload, err
}

func jsonStringField(t *testing.T, rec *httptest.ResponseRecorder, key string) string {
	var (
		m   map[string]any
		raw string
		ok  bool
	)
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	raw, ok = m[key].(string)
	if !ok || raw == "" {
		t.Fatalf("missing %q in %s", key, rec.Body.String())
	}
	return raw
}
