package usecase

import (
	"context"
	"encoding/json"

	"crowdfund-app/entity"

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
	return p.fulfillPaymentIntent(ctx, pi)
}

func (p *PaymentUseCase) onInvoicePaid(ctx context.Context, event stripesdk.Event) error {
	var (
		inv    stripesdk.Invoice
		exists *entity.Purchase
		err    error
	)
	err = json.Unmarshal(event.Data.Raw, &inv)
	if err != nil {
		return err
	}
	exists, err = p.Purchases.FindStripeInvoiceID(ctx, inv.ID)
	if err != nil {
		return err
	}
	if exists != nil {
		return nil
	}
	if isAutoSubscriptionRenewal(inv) {
		return p.fulfillAutoRenewal(ctx, inv)
	}
	return p.fulfillInvoiceFirstSupport(ctx, inv)
}
