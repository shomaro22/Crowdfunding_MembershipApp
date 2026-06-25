package usecase

import (
	"context"
	"encoding/json"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
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
	var exists bool
	exists, err = p.Webhooks.Exists(ctx, event.ID)
	if err != nil {
		return err
	}
	if exists {
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
		return err
	}
	return p.Webhooks.Create(ctx, &entity.WebhookEvent{
		ID:            uuid.New(),
		StripeEventID: event.ID,
		EventType:     string(event.Type),
		ProcessedAt:   time.Now().UTC(),
	})
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
