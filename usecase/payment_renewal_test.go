package usecase

import (
	"testing"
	"time"

	"crowdfund-app/model"
	"crowdfund-app/stripe"
)

func TestIsAutoSubscriptionRenewal(t *testing.T) {
	var cases = []struct {
		reason string
		want   bool
	}{
		{stripe.InvoiceBillingReasonSubscriptionCycle, true},
		{stripe.InvoiceBillingReasonSubscriptionUpdate, true},
		{"subscription_create", false},
		{"manual", false},
	}
	for _, tc := range cases {
		if got := stripe.IsAutoSubscriptionRenewal(tc.reason); got != tc.want {
			t.Fatalf("billing_reason=%s: got %v want %v", tc.reason, got, tc.want)
		}
	}
}

func TestCanIssueKonbiniRenewal(t *testing.T) {
	var (
		periodEnd = time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
		s         = &model.ProjectSupporter{
			Status:           model.SupporterStatusActive,
			BillingMode:      model.BillingModeKonbini,
			CurrentPeriodEnd: periodEnd,
		}
	)
	if canIssueKonbiniRenewal(s, periodEnd.AddDate(0, 0, -8)) {
		t.Fatal("too early should be denied")
	}
	if !canIssueKonbiniRenewal(s, periodEnd.AddDate(0, 0, -7)) {
		t.Fatal("7 days before end should be allowed")
	}
	s.Status = model.SupporterStatusPendingRenewal
	if !canIssueKonbiniRenewal(s, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("pending renewal should always be allowed")
	}
}

func TestInvoicePurchaseID_FromSubscriptionMetadata(t *testing.T) {
	var inv stripe.InvoiceSnapshot
	inv.SubscriptionMeta = map[string]string{"purchase_id": "abc-123"}
	if got := inv.PurchaseID(); got != "abc-123" {
		t.Fatalf("got %q", got)
	}
}
