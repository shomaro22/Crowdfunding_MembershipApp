//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"crowdfund-app/model"
	"crowdfund-app/stripe"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type connectAccountResponse struct {
	Account *struct {
		StripeAccountID  string `json:"stripe_account_id"`
		ChargesEnabled   bool   `json:"charges_enabled"`
		PayoutsEnabled   bool   `json:"payouts_enabled"`
		DetailsSubmitted bool   `json:"details_submitted"`
	} `json:"account"`
}

type connectOnboardingResponse struct {
	URL string `json:"url"`
}

func TestConnectGetAccountUnlinked(t *testing.T) {
	var (
		e       *echo.Echo
		gdb     *gorm.DB
		seller  *model.User
		tok     string
		rec     *httptest.ResponseRecorder
		payload connectAccountResponse
		err     error
	)
	e, _ = setupPaymentApp(t)
	gdb = testDB(t)
	seller = seedUser(t, gdb, seedOpts{Role: model.UserRoleSeller, TOTPEnabled: true})
	tok, _ = issueToken(seller, true)

	rec = httpGET(e, "/seller/connect", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	err = json.Unmarshal(rec.Body.Bytes(), &payload)
	if err != nil {
		t.Fatal(err)
	}
	if payload.Account != nil {
		t.Fatalf("expected null account, got %+v", payload.Account)
	}
}

func TestConnectOnboardingCreatesAccount(t *testing.T) {
	var (
		e       *echo.Echo
		stub    *stripe.StubClient
		gdb     *gorm.DB
		seller  *model.User
		tok     string
		rec     *httptest.ResponseRecorder
		onboard connectOnboardingResponse
		after   connectAccountResponse
		err     error
	)
	e, stub = setupPaymentApp(t)
	_ = stub
	gdb = testDB(t)
	seller = seedUser(t, gdb, seedOpts{Role: model.UserRoleSeller, TOTPEnabled: true})
	tok, _ = issueToken(seller, true)

	rec = httpPOST(e, "/seller/connect/onboarding", tok, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("onboarding status=%d body=%s", rec.Code, rec.Body.String())
	}
	err = json.Unmarshal(rec.Body.Bytes(), &onboard)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(onboard.URL, "http://") && !strings.HasPrefix(onboard.URL, "https://") {
		t.Fatalf("invalid onboarding url=%q", onboard.URL)
	}

	rec = httpGET(e, "/seller/connect", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", rec.Code, rec.Body.String())
	}
	err = json.Unmarshal(rec.Body.Bytes(), &after)
	if err != nil {
		t.Fatal(err)
	}
	if after.Account == nil || after.Account.StripeAccountID == "" {
		t.Fatalf("expected linked account, body=%s", rec.Body.String())
	}
	if !after.Account.ChargesEnabled || !after.Account.PayoutsEnabled {
		t.Fatalf("connect flags not enabled: %+v", after.Account)
	}
}

func TestConnectBuyerForbidden(t *testing.T) {
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

	rec = httpGET(e, "/seller/connect", tok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rec.Code)
	}
}
