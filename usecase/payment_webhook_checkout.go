package usecase

import (
	"context"
	"encoding/json"

	"crowdfund-app/entity"

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
	purchase, err = p.Purchases.FindBySessionID(ctx, sess.ID)
	if err != nil || purchase == nil || purchase.Status != entity.PurchaseStatusPending {
		return err
	}
	return p.markPurchaseFailed(ctx, purchase, "checkout session expired")
}
