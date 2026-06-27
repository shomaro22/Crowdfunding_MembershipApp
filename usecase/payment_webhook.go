package usecase

import (
	"context"

	"crowdfund-app/entity"
	"crowdfund-app/stripe"
)

func (p *PaymentUseCase) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
	var (
		event stripe.WebhookEvent
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
	claimed, err = p.Webhooks.Claim(ctx, event.ID, event.Type)
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}
	switch event.Type {
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
	p.logPaymentWebhook(ctx, event.Type, event.ID)
	return nil
}

func (p *PaymentUseCase) onAccountUpdated(ctx context.Context, event stripe.WebhookEvent) error {
	var (
		acct stripe.AccountEvent
		err  error
	)
	if p.ConnectSync == nil {
		return nil
	}
	acct, err = stripe.ParseWebhookAccount(event)
	if err != nil {
		return err
	}
	_, err = p.ConnectSync.SyncFromStripeAccountID(ctx, acct.AccountID)
	return err
}

func (p *PaymentUseCase) onCheckoutCompleted(ctx context.Context, event stripe.WebhookEvent) error {
	var (
		sess stripe.CheckoutSessionSnapshot
		err  error
	)
	sess, err = stripe.ParseWebhookCheckoutSession(event)
	if err != nil {
		return err
	}
	if isKonbiniSession(sess) {
		return nil
	}
	if sess.PaymentStatus != stripe.CheckoutPaymentStatusPaid {
		return nil
	}
	return p.fulfillCheckoutSession(ctx, sess)
}

func (p *PaymentUseCase) onCheckoutAsyncPaid(ctx context.Context, event stripe.WebhookEvent) error {
	var (
		sess stripe.CheckoutSessionSnapshot
		err  error
	)
	sess, err = stripe.ParseWebhookCheckoutSession(event)
	if err != nil {
		return err
	}
	return p.fulfillCheckoutSession(ctx, sess)
}

func (p *PaymentUseCase) onCheckoutAsyncFailed(ctx context.Context, event stripe.WebhookEvent) error {
	var (
		sess     stripe.CheckoutSessionSnapshot
		purchase *entity.Purchase
		err      error
	)
	sess, err = stripe.ParseWebhookCheckoutSession(event)
	if err != nil {
		return err
	}
	purchase, err = p.Purchases.FindSessionID(ctx, sess.ID)
	if err != nil {
		return err
	}
	return p.handleKonbiniCheckoutUnpaid(ctx, purchase, "async payment failed", false)
}

func (p *PaymentUseCase) onCheckoutExpired(ctx context.Context, event stripe.WebhookEvent) error {
	var (
		sess     stripe.CheckoutSessionSnapshot
		purchase *entity.Purchase
		err      error
	)
	sess, err = stripe.ParseWebhookCheckoutSession(event)
	if err != nil {
		return err
	}
	purchase, err = p.Purchases.FindSessionID(ctx, sess.ID)
	if err != nil {
		return err
	}
	return p.handleKonbiniCheckoutUnpaid(ctx, purchase, "checkout session expired", true)
}

func isKonbiniSession(sess stripe.CheckoutSessionSnapshot) bool {
	var i int
	for i = range sess.PaymentMethodTypes {
		if sess.PaymentMethodTypes[i] == "konbini" {
			return true
		}
	}
	return parseCheckoutPurpose(sess.Metadata) == checkoutKonbiniRenewal ||
		sess.Metadata[stripe.MetaBillingMode] == string(entity.BillingModeKonbini)
}

func (p *PaymentUseCase) onPaymentIntentSucceeded(ctx context.Context, event stripe.WebhookEvent) error {
	var (
		pi  stripe.PaymentIntentDetail
		err error
	)
	pi, err = stripe.ParseWebhookPaymentIntent(event)
	if err != nil {
		return err
	}
	return p.fulfillPaymentIntent(ctx, pi)
}

func (p *PaymentUseCase) onInvoicePaid(ctx context.Context, event stripe.WebhookEvent) error {
	var (
		inv    stripe.InvoiceSnapshot
		exists *entity.Purchase
		err    error
	)
	inv, err = stripe.ParseWebhookInvoice(event)
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
	if stripe.IsAutoSubscriptionRenewal(inv.BillingReason) {
		return p.fulfillAutoRenewal(ctx, inv)
	}
	return p.fulfillInvoiceFirstSupport(ctx, inv)
}
