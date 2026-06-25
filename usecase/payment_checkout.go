package usecase

import (
	"context"

	"crowdfund-app/entity"
	"crowdfund-app/stripe"
)

func (p *PaymentUseCase) CreateCheckout(ctx context.Context, in CheckoutIn) (CheckoutOut, error) {
	var (
		sc       supportContext
		purchase *entity.Purchase
		meta     map[string]string
		sess     stripe.CheckoutSessionOut
		pm       entity.PaymentMethod
		err      error
	)
	if p.Stripe == nil {
		return CheckoutOut{}, ErrStripeUnavailable
	}
	pm = in.PaymentMethod
	if pm == "" {
		pm = entity.PaymentMethodKonbini
	}
	if pm != entity.PaymentMethodKonbini && pm != entity.PaymentMethodCard {
		return CheckoutOut{}, ErrBadRequest
	}
	sc, err = p.loadSupportContext(ctx, in.BuyerID, in.ProjectID, in.PlanID)
	if err != nil {
		return CheckoutOut{}, err
	}
	purchase, err = p.createPendingPurchase(ctx, sc, pm, entity.BillingModeKonbini, nil, nil)
	if err != nil {
		return CheckoutOut{}, err
	}
	meta = paymentMetadata(purchase.ID, in.ProjectID, in.PlanID, in.BuyerID, entity.BillingModeKonbini)
	sess, err = p.Stripe.CreateCheckoutSession(ctx, stripe.CheckoutSessionIn{
		ProductName:      sc.project.Title,
		Amount:           int64(sc.plan.Price),
		ConnectAccountID: sc.connect.StripeAccountID,
		CustomerEmail:    sc.buyer.Email,
		SuccessURL:       checkoutURL(in.SuccessURL, p.Stripe.AppURL()+"/support/success?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:        checkoutURL(in.CancelURL, p.Stripe.AppURL()+"/support/cancel"),
		PaymentMethod:    string(pm),
		Metadata:         meta,
	})
	if err != nil {
		return CheckoutOut{}, err
	}
	purchase.StripeCheckoutID = &sess.SessionID
	purchase.UpdatedAt = sc.now
	if err = p.Purchases.Update(ctx, purchase); err != nil {
		return CheckoutOut{}, err
	}
	return CheckoutOut{SessionID: sess.SessionID, URL: sess.URL}, nil
}
