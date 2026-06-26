package usecase

import (
	"testing"
	"time"

	"crowdfund-app/entity"

	stripesdk "github.com/stripe/stripe-go/v82"
)

func TestIsAutoSubscriptionRenewal(t *testing.T) {
	var cases = []struct {
		reason stripesdk.InvoiceBillingReason
		want   bool
	}{
		{stripesdk.InvoiceBillingReasonSubscriptionCycle, true},
		{stripesdk.InvoiceBillingReasonSubscriptionUpdate, true},
		{stripesdk.InvoiceBillingReasonSubscriptionCreate, false},
		{stripesdk.InvoiceBillingReasonManual, false},
	}
	for _, tc := range cases {
		inv := stripesdk.Invoice{BillingReason: tc.reason}
		if got := isAutoSubscriptionRenewal(inv); got != tc.want {
			t.Fatalf("billing_reason=%s: got %v want %v", tc.reason, got, tc.want)
		}
	}
}

func TestCanIssueKonbiniRenewal(t *testing.T) {
	var (
		periodEnd = time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
		s         = &entity.ProjectSupporter{
			Status:           entity.SupporterStatusActive,
			BillingMode:      entity.BillingModeKonbini,
			CurrentPeriodEnd: periodEnd,
		}
	)
	if canIssueKonbiniRenewal(s, periodEnd.AddDate(0, 0, -8)) {
		t.Fatal("too early should be denied")
	}
	if !canIssueKonbiniRenewal(s, periodEnd.AddDate(0, 0, -7)) {
		t.Fatal("7 days before end should be allowed")
	}
	s.Status = entity.SupporterStatusPendingRenewal
	if !canIssueKonbiniRenewal(s, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("pending renewal should always be allowed")
	}
}

func TestInvoicePurchaseID_FromSubscriptionMetadata(t *testing.T) {
	var inv stripesdk.Invoice
	inv.Parent = &stripesdk.InvoiceParent{
		SubscriptionDetails: &stripesdk.InvoiceParentSubscriptionDetails{
			Metadata: map[string]string{"purchase_id": "abc-123"},
		},
	}
	if got := invoicePurchaseID(inv); got != "abc-123" {
		t.Fatalf("got %q", got)
	}
}
