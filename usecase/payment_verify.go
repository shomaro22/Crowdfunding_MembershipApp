package usecase

import (
	"context"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	stripesdk "github.com/stripe/stripe-go/v82"
)

func (p *PaymentUseCase) VerifyCheckout(ctx context.Context, buyerID uuid.UUID, sessionID string) (SupporterView, error) {
	var (
		sess     *stripesdk.CheckoutSession
		purchase *entity.Purchase
		err      error
	)
	if p.Stripe == nil {
		return SupporterView{}, ErrStripeUnavailable
	}
	if sessionID == "" {
		return SupporterView{}, ErrBadRequest
	}
	sess, err = p.Stripe.GetCheckoutSession(ctx, sessionID)
	if err != nil {
		return SupporterView{}, ErrBadRequest
	}
	purchase, err = p.Purchases.FindSessionID(ctx, sessionID)
	if err != nil || purchase == nil {
		return SupporterView{}, ErrNotFound
	}
	if err = verifyPurchaseBuyer(purchase, buyerID); err != nil {
		return SupporterView{}, err
	}
	if purchase.Status == entity.PurchaseStatusPaid {
		return p.supporterViewForPurchase(ctx, purchase)
	}
	if sess.PaymentStatus != stripesdk.CheckoutSessionPaymentStatusPaid {
		return SupporterView{}, ErrBadRequest
	}
	if err = p.fulfillCheckoutSession(ctx, *sess); err != nil {
		return SupporterView{}, err
	}
	purchase, err = p.Purchases.FindID(ctx, purchase.ID)
	if err != nil || purchase == nil {
		return SupporterView{}, ErrNotFound
	}
	return p.supporterViewForPurchase(ctx, purchase)
}

func (p *PaymentUseCase) VerifyIntent(ctx context.Context, buyerID uuid.UUID, intentID string) (SupporterView, error) {
	var (
		pi       *stripesdk.PaymentIntent
		purchase *entity.Purchase
		err      error
	)
	if p.Stripe == nil {
		return SupporterView{}, ErrStripeUnavailable
	}
	if intentID == "" {
		return SupporterView{}, ErrBadRequest
	}
	pi, err = p.Stripe.GetPaymentIntentDetail(ctx, intentID)
	if err != nil {
		return SupporterView{}, ErrBadRequest
	}
	purchase, err = p.Purchases.FindIntentID(ctx, intentID)
	if err != nil || purchase == nil {
		return SupporterView{}, ErrNotFound
	}
	if err = verifyPurchaseBuyer(purchase, buyerID); err != nil {
		return SupporterView{}, err
	}
	if purchase.Status == entity.PurchaseStatusPaid {
		return p.supporterViewForPurchase(ctx, purchase)
	}
	if pi.Status != stripesdk.PaymentIntentStatusSucceeded {
		return SupporterView{}, ErrBadRequest
	}
	if err = p.fulfillPaymentIntent(ctx, *pi); err != nil {
		return SupporterView{}, err
	}
	purchase, err = p.Purchases.FindID(ctx, purchase.ID)
	if err != nil || purchase == nil {
		return SupporterView{}, ErrNotFound
	}
	return p.supporterViewForPurchase(ctx, purchase)
}

func verifyPurchaseBuyer(purchase *entity.Purchase, buyerID uuid.UUID) error {
	if purchase == nil || purchase.BuyerID != buyerID {
		return ErrForbidden
	}
	return nil
}

func (p *PaymentUseCase) supporterViewForPurchase(ctx context.Context, purchase *entity.Purchase) (SupporterView, error) {
	var (
		supporter *entity.ProjectSupporter
		err       error
	)
	if purchase == nil {
		return SupporterView{}, ErrNotFound
	}
	if purchase.SupporterID != nil {
		supporter, err = p.Supporters.FindID(ctx, *purchase.SupporterID)
	} else {
		supporter, err = p.Supporters.FindActive(ctx, purchase.ProjectID, purchase.BuyerID)
	}
	if err != nil || supporter == nil {
		return SupporterView{}, ErrNotFound
	}
	uc := &SupporterUseCase{Projects: p.Projects, Supporters: p.Supporters}
	return uc.enrichView(ctx, supporter)
}
