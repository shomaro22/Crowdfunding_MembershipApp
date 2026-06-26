package usecase

import (
	"context"
	"encoding/json"

	stripesdk "github.com/stripe/stripe-go/v82"
)

func (p *PaymentUseCase) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
	var (
		event stripesdk.Event
		err   error
	)
	if p.Stripe == nil {
		return ErrStripeUnavailable
	}
	event, err = p.Stripe.ConstructWebhookEvent(payload, signature)
	if err != nil {
		return err
	}
	var claimed bool
	claimed, err = p.Webhooks.Claim(ctx, event.ID, string(event.Type))
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}
	switch string(event.Type) {
	case "checkout.session.completed":
		err = p.onCheckoutCompleted(ctx, event)
	case "checkout.session.async_payment_succeeded":
		err = p.onCheckoutAsyncPaid(ctx, event)
	case "checkout.session.async_payment_failed":
		err = p.onCheckoutAsyncFailed(ctx, event)
	case "checkout.session.expired":
		err = p.onCheckoutExpired(ctx, event)
	case "payment_intent.succeeded":
		err = p.onPaymentIntentSucceeded(ctx, event)
	case "invoice.paid":
		err = p.onInvoicePaid(ctx, event)
	case "account.updated":
		err = p.onAccountUpdated(ctx, event)
	default:
		err = nil
	}
	if err != nil {
		_ = p.Webhooks.Release(ctx, event.ID)
		return err
	}
	p.logPaymentWebhook(ctx, string(event.Type), event.ID)
	return nil
}

func (p *PaymentUseCase) onAccountUpdated(ctx context.Context, event stripesdk.Event) error {
	var (
		acct stripesdk.Account
		err  error
	)
	if p.ConnectUC == nil {
		return nil
	}
	err = json.Unmarshal(event.Data.Raw, &acct)
	if err != nil {
		return err
	}
	_, err = p.ConnectUC.SyncFromStripeAccountID(ctx, acct.ID)
	return err
}
