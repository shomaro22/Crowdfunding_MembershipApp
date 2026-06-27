package usecase

import (
	"context"
	"encoding/json"

	"crowdfund-app/entity"
	"crowdfund-app/stripe"

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
	if p.ConnectSync == nil {
		return nil
	}
	err = json.Unmarshal(event.Data.Raw, &acct)
	if err != nil {
		return err
	}
	_, err = p.ConnectSync.SyncFromStripeAccountID(ctx, acct.ID)
	return err
}

func (p *PaymentUseCase) onCheckoutCompleted(ctx context.Context, event stripesdk.Event) error {
	var (
		sess stripesdk.CheckoutSession
		err  error
	)
	err = json.Unmarshal(event.Data.Raw, &sess)
	if err != nil {
		return err
	}
	if isKonbiniSession(sess) {
		return nil
	}
	if sess.PaymentStatus != stripesdk.CheckoutSessionPaymentStatusPaid {
		return nil
	}
	return p.fulfillCheckoutSession(ctx, sess)
}

func (p *PaymentUseCase) onCheckoutAsyncPaid(ctx context.Context, event stripesdk.Event) error {
	var (
		sess stripesdk.CheckoutSession
		err  error
	)
	err = json.Unmarshal(event.Data.Raw, &sess)
	if err != nil {
		return err
	}
	return p.fulfillCheckoutSession(ctx, sess)
}

func (p *PaymentUseCase) onCheckoutAsyncFailed(ctx context.Context, event stripesdk.Event) error {
	var (
		sess     stripesdk.CheckoutSession
		purchase *entity.Purchase
		err      error
	)
	err = json.Unmarshal(event.Data.Raw, &sess)
	if err != nil {
		return err
	}
	purchase, err = p.Purchases.FindSessionID(ctx, sess.ID)
	if err != nil {
		return err
	}
	return p.handleKonbiniCheckoutUnpaid(ctx, purchase, "async payment failed", false)
}

func (p *PaymentUseCase) onCheckoutExpired(ctx context.Context, event stripesdk.Event) error {
	var (
		sess     stripesdk.CheckoutSession
		purchase *entity.Purchase
		err      error
	)
	err = json.Unmarshal(event.Data.Raw, &sess)
	if err != nil {
		return err
	}
	purchase, err = p.Purchases.FindSessionID(ctx, sess.ID)
	if err != nil {
		return err
	}
	return p.handleKonbiniCheckoutUnpaid(ctx, purchase, "checkout session expired", true)
}

func isKonbiniSession(sess stripesdk.CheckoutSession) bool {
	var i int
	for i = range sess.PaymentMethodTypes {
		if sess.PaymentMethodTypes[i] == "konbini" {
			return true
		}
	}
	return parseCheckoutPurpose(sess.Metadata) == checkoutKonbiniRenewal ||
		sess.Metadata[stripe.MetaBillingMode] == string(entity.BillingModeKonbini)
}

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
