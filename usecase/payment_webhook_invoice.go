package usecase

import (
	"context"
	"encoding/json"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/stripe"

	"github.com/google/uuid"
	stripesdk "github.com/stripe/stripe-go/v82"
)

func (p *PaymentUseCase) onPaymentIntentSucceeded(ctx context.Context, event stripesdk.Event) error {
	var (
		pi  stripesdk.PaymentIntent
		err error
	)
	err = json.Unmarshal(event.Data.Raw, &pi)
	if err != nil {
		return err
	}
	if pi.Metadata[stripe.MetaBillingMode] != string(entity.BillingModeAuto) {
		return nil
	}
	return p.fulfillPaymentIntent(ctx, pi)
}

func (p *PaymentUseCase) onInvoicePaid(ctx context.Context, event stripesdk.Event) error {
	var (
		inv stripesdk.Invoice
		err error
	)
	err = json.Unmarshal(event.Data.Raw, &inv)
	if err != nil {
		return err
	}
	var subDetails *stripesdk.InvoiceParentSubscriptionDetails
	if inv.Parent != nil {
		subDetails = inv.Parent.SubscriptionDetails
	}
	if inv.Metadata[stripe.MetaPurchaseID] == "" && (subDetails == nil || subDetails.Metadata == nil) {
		return nil
	}
	var purchaseIDStr string
	if inv.Metadata[stripe.MetaPurchaseID] != "" {
		purchaseIDStr = inv.Metadata[stripe.MetaPurchaseID]
	} else {
		purchaseIDStr = subDetails.Metadata[stripe.MetaPurchaseID]
	}
	var (
		start = time.Unix(inv.PeriodStart, 0).UTC()
		end   = time.Unix(inv.PeriodEnd, 0).UTC()
		cust  *string
		sub   *string
		invID *string
	)
	if inv.Customer != nil {
		cust = stripeString(inv.Customer.ID)
	}
	if subDetails != nil && subDetails.Subscription != nil {
		sub = stripeString(subDetails.Subscription.ID)
	}
	invID = stripeString(inv.ID)
	return p.fulfillPurchaseID(ctx, uuid.MustParse(purchaseIDStr), start, end, cust, sub, invID)
}
