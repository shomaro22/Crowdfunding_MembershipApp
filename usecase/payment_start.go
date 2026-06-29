package usecase

import (
	"context"

	"crowdfund-app/model"
	"crowdfund-app/stripe"

	"github.com/google/uuid"
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
		return CheckoutOut{}, err
	}
	purchase.StripeCheckoutID = &sess.SessionID
	if sess.PaymentIntentID != "" {
		purchase.StripeIntentID = &sess.PaymentIntentID
	}
	purchase.UpdatedAt = sc.now
	if err = p.savePendingAfterStripe(ctx, purchase); err != nil {
		return CheckoutOut{}, err
	}
	p.logPaymentCheckoutStarted(ctx, purchase, in.Meta, "checkout")
	p.logPurchaseCreated(ctx, purchase, in.Meta)
	return CheckoutOut{SessionID: sess.SessionID, URL: sess.URL}, nil
}

func (p *PaymentUseCase) CreateIntent(ctx context.Context, in IntentIn) (IntentOut, error) {
	var (
		sc       supportContext
		purchase model.Purchase
		meta     map[string]string
		sub      stripe.SubscriptionIntentOut
		err      error
	)
	if p.Stripe == nil {
		return IntentOut{}, ErrStripeUnavailable
	}
	sc, err = p.loadSupportContext(ctx, in.BuyerID, in.ProjectID, in.PlanID)
	if err != nil {
		return IntentOut{}, err
	}
	purchase = p.newPendingPurchase(sc, model.PaymentMethodCard, nil, nil)
	meta = paymentMetadata(purchase.ID, in.ProjectID, in.PlanID, in.BuyerID, model.BillingModeAuto)
	sub, err = p.Stripe.CreateSubscriptionIntent(ctx, stripe.SubscriptionIntentIn{
		ProductName:      sc.project.Title,
		Amount:           int64(sc.plan.Price),
		ConnectAccountID: sc.connect.StripeAccountID,
		CustomerEmail:    sc.buyer.Email,
		Metadata:         meta,
	})
	if err != nil {
		return IntentOut{}, err
	}
	purchase.StripeIntentID = &sub.PaymentIntentID
	purchase.UpdatedAt = sc.now
	if err = p.savePendingAfterStripe(ctx, purchase); err != nil {
		return IntentOut{}, err
	}
	p.logPaymentCheckoutStarted(ctx, purchase, in.Meta, "intent")
	p.logPurchaseCreated(ctx, purchase, in.Meta)
	return IntentOut{ClientSecret: sub.ClientSecret, PaymentIntentID: sub.PaymentIntentID}, nil
}

func (p *PaymentUseCase) VerifyCheckout(ctx context.Context, buyerID uuid.UUID, sessionID string) (SupporterView, error) {
	var (
		sess     stripe.CheckoutSessionSnapshot
		purchase model.Purchase
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
	purchase, err = p.Purchases.GetSessionID(ctx, sessionID)
	if err != nil {
		return SupporterView{}, mapRepoNotFound(err)
	}
	return p.verifyAndFulfill(ctx, buyerID, purchase, sess.PaymentStatus == stripe.CheckoutPaymentStatusPaid, func(ctx context.Context) error {
		return p.fulfillCheckoutSession(ctx, sess)
	})
}

func (p *PaymentUseCase) VerifyIntent(ctx context.Context, buyerID uuid.UUID, intentID string) (SupporterView, error) {
	var (
		pi       stripe.PaymentIntentDetail
		purchase model.Purchase
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
	purchase, err = p.Purchases.GetIntentID(ctx, intentID)
	if err != nil {
		return SupporterView{}, mapRepoNotFound(err)
	}
	return p.verifyAndFulfill(ctx, buyerID, purchase, pi.Status == stripe.PaymentIntentStatusSucceeded, func(ctx context.Context) error {
		return p.fulfillPaymentIntent(ctx, pi)
	})
}

func (p *PaymentUseCase) verifyAndFulfill(
	ctx context.Context,
	buyerID uuid.UUID,
	purchase model.Purchase,
	paid bool,
	fulfill func(context.Context) error,
) (SupporterView, error) {
	var (
		refreshed model.Purchase
		err       error
	)
	if err = verifyPurchaseBuyer(purchase, buyerID); err != nil {
		return SupporterView{}, err
	}
	if purchase.Status == model.PurchaseStatusPaid {
		return p.supporterViewForPurchase(ctx, purchase)
	}
	if !paid {
		return SupporterView{}, ErrBadRequest
	}
	if err = fulfill(ctx); err != nil {
		return SupporterView{}, err
	}
	refreshed, err = p.Purchases.GetID(ctx, purchase.ID)
	if err != nil {
		return SupporterView{}, mapRepoNotFound(err)
	}
	return p.supporterViewForPurchase(ctx, refreshed)
}

func verifyPurchaseBuyer(purchase model.Purchase, buyerID uuid.UUID) error {
	if purchase.BuyerID != buyerID {
		return ErrForbidden
	}
	return nil
}

func (p *PaymentUseCase) supporterViewForPurchase(ctx context.Context, purchase model.Purchase) (SupporterView, error) {
	var (
		supporter model.ProjectSupporter
		err       error
	)
	if purchase.SupporterID != nil {
		supporter, err = p.Supporters.GetID(ctx, *purchase.SupporterID)
	} else {
		supporter, err = p.Supporters.GetActive(ctx, purchase.ProjectID, purchase.BuyerID)
	}
	if err != nil {
		return SupporterView{}, mapRepoNotFound(err)
	}
	if p.SupporterViewer == nil {
		return SupporterView{}, ErrNotFound
	}
	return p.SupporterViewer.ViewForSupporter(ctx, supporter)
}
