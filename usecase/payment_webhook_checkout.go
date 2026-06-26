package usecase

import (
	"context"
	"encoding/json"

	"crowdfund-app/entity"
	"crowdfund-app/stripe"

	stripesdk "github.com/stripe/stripe-go/v82"
)

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
