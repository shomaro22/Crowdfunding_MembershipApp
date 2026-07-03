package usecase

import (
	"context"

	"crowdfund-app/model"
	"crowdfund-app/stripe"
)

func (p *PaymentUseCase) CreateCheckout(ctx context.Context, in CheckoutIn) (CheckoutOut, error) {
	var (
		sc       supportContext
		purchase model.Purchase
		meta     map[string]string
		sess     stripe.CheckoutSessionOut
		pm       model.PaymentMethod
		err      error
	)
	if p.Stripe == nil {
		return CheckoutOut{}, ErrStripeUnavailable
	}
	pm = model.PaymentMethod(in.PaymentMethod)
	if pm == "" {
		pm = model.PaymentMethodKonbini
	}
	if pm != model.PaymentMethodKonbini && pm != model.PaymentMethodCard {
		return CheckoutOut{}, ErrBadRequest
	}
	sc, err = p.loadCheckoutContext(ctx, in.BuyerID, in.ProjectID, in.PlanID, pm)
	if err != nil {
		return CheckoutOut{}, err
	}
	purchase = p.newPendingPurchase(sc, pm, nil, nil)
	if err = p.persistPendingPurchase(ctx, purchase); err != nil {
		return CheckoutOut{}, err
	}
	meta = checkoutMetadata(sc, purchase.ID, pm)
	sess, err = p.Stripe.CreateCheckoutSession(ctx, stripe.CheckoutSessionIn{
		ProductName:      sc.project.Title,
		Amount:           int64(sc.plan.Price),
		ConnectAccountID: sc.connect.StripeAccountID,
		CustomerEmail:    sc.buyer.Email,
		SuccessURL:       checkoutURL(in.SuccessURL, in.FallbackSuccessURL),
		CancelURL:        checkoutURL(in.CancelURL, in.FallbackCancelURL),
		PaymentMethod:    string(pm),
		Metadata:         meta,
	})
	if err != nil {
		_ = p.markPurchaseCancelled(ctx, purchase, "stripe_checkout_failed")
		return CheckoutOut{}, err
	}
	purchase.StripeCheckoutID = &sess.SessionID
	if sess.PaymentIntentID != "" {
		purchase.StripeIntentID = &sess.PaymentIntentID
	}
	purchase.UpdatedAt = sc.now
	if err = p.Purchases.Update(ctx, &purchase); err != nil {
		return CheckoutOut{}, err
	}
	p.logPaymentCheckoutStarted(ctx, purchase, in.Meta, "checkout")
	p.logPurchaseCreated(ctx, purchase, in.Meta)
	return CheckoutOut{SessionID: sess.SessionID, URL: sess.URL}, nil
}
