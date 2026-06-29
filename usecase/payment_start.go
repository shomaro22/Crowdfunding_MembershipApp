package usecase

import (
	"context"

	"crowdfund-app/entity"

	"github.com/google/uuid"
)

func (p *PaymentUseCase) CreateCheckout(ctx context.Context, in CheckoutIn) (CheckoutOut, error) {
	var (
		sc         supportContext
		purchase   *entity.Purchase
		meta       map[string]string
		sess       CheckoutSessionOut
		successURL string
		cancelURL  string
		pm         entity.PaymentMethod
		err        error
	)
	if p.PaymentGateway == nil {
		return CheckoutOut{}, ErrStripeUnavailable
	}
	pm = entity.PaymentMethod(in.PaymentMethod)
	if pm == "" {
		pm = entity.PaymentMethodKonbini
	}
	if pm != entity.PaymentMethodKonbini && pm != entity.PaymentMethodCard {
		return CheckoutOut{}, ErrBadRequest
	}
	sc, err = p.loadCheckoutContext(ctx, in.BuyerID, in.ProjectID, in.PlanID, pm)
	if err != nil {
		return CheckoutOut{}, err
	}
	purchase = p.newPendingPurchase(sc, pm, nil, nil)
	meta = checkoutMetadata(sc, purchase.ID, pm)
	successURL, err = resolveCheckoutURL(in.SuccessURL, in.FallbackSuccessURL, p.AppPublicURL)
	if err != nil {
		return CheckoutOut{}, err
	}
	cancelURL, err = resolveCheckoutURL(in.CancelURL, in.FallbackCancelURL, p.AppPublicURL)
	if err != nil {
		return CheckoutOut{}, err
	}
	sess, err = p.PaymentGateway.CreateCheckoutSession(ctx, CheckoutSessionIn{
		ProductName:      sc.project.Title,
		Amount:           int64(sc.plan.Price),
		ConnectAccountID: sc.connect.StripeAccountID,
		CustomerEmail:    sc.buyer.Email,
		SuccessURL:       successURL,
		CancelURL:        cancelURL,
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
		purchase *entity.Purchase
		meta     map[string]string
		sub      SubscriptionIntentOut
		err      error
	)
	if p.PaymentGateway == nil {
		return IntentOut{}, ErrStripeUnavailable
	}
	sc, err = p.loadSupportContext(ctx, in.BuyerID, in.ProjectID, in.PlanID)
	if err != nil {
		return IntentOut{}, err
	}
	purchase = p.newPendingPurchase(sc, entity.PaymentMethodCard, nil, nil)
	meta = paymentMetadata(purchase.ID, in.ProjectID, in.PlanID, in.BuyerID, entity.BillingModeAuto)
	sub, err = p.PaymentGateway.CreateSubscriptionIntent(ctx, SubscriptionIntentIn{
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
		sess     CheckoutSessionSnapshot
		purchase *entity.Purchase
		err      error
	)
	if p.PaymentGateway == nil {
		return SupporterView{}, ErrStripeUnavailable
	}
	if sessionID == "" {
		return SupporterView{}, ErrBadRequest
	}
	sess, err = p.PaymentGateway.GetCheckoutSession(ctx, sessionID)
	if err != nil {
		return SupporterView{}, ErrBadRequest
	}
	purchase, err = p.Purchases.FindSessionID(ctx, sessionID)
	if err != nil {
		return SupporterView{}, mapRepoNotFound(err)
	}
	return p.verifyAndFulfill(ctx, buyerID, purchase, sess.PaymentStatus == CheckoutPaymentStatusPaid, func(ctx context.Context) error {
		return p.fulfillCheckoutSession(ctx, sess)
	})
}

func (p *PaymentUseCase) VerifyIntent(ctx context.Context, buyerID uuid.UUID, intentID string) (SupporterView, error) {
	var (
		pi       PaymentIntentDetail
		purchase *entity.Purchase
		err      error
	)
	if p.PaymentGateway == nil {
		return SupporterView{}, ErrStripeUnavailable
	}
	if intentID == "" {
		return SupporterView{}, ErrBadRequest
	}
	pi, err = p.PaymentGateway.GetPaymentIntentDetail(ctx, intentID)
	if err != nil {
		return SupporterView{}, ErrBadRequest
	}
	purchase, err = p.Purchases.FindIntentID(ctx, intentID)
	if err != nil {
		return SupporterView{}, mapRepoNotFound(err)
	}
	return p.verifyAndFulfill(ctx, buyerID, purchase, pi.Status == PaymentIntentStatusSucceeded, func(ctx context.Context) error {
		return p.fulfillPaymentIntent(ctx, pi)
	})
}

func (p *PaymentUseCase) verifyAndFulfill(
	ctx context.Context,
	buyerID uuid.UUID,
	purchase *entity.Purchase,
	paid bool,
	fulfill func(context.Context) error,
) (SupporterView, error) {
	var (
		refreshed *entity.Purchase
		err       error
	)
	if err = verifyPurchaseBuyer(purchase, buyerID); err != nil {
		return SupporterView{}, err
	}
	if purchase.Status == entity.PurchaseStatusPaid {
		return p.supporterViewForPurchase(ctx, purchase)
	}
	if !paid {
		return SupporterView{}, ErrBadRequest
	}
	if err = fulfill(ctx); err != nil {
		return SupporterView{}, err
	}
	refreshed, err = p.Purchases.FindID(ctx, purchase.ID)
	if err != nil {
		return SupporterView{}, mapRepoNotFound(err)
	}
	return p.supporterViewForPurchase(ctx, refreshed)
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
	if err != nil {
		return SupporterView{}, mapRepoNotFound(err)
	}
	if p.SupporterViewer == nil {
		return SupporterView{}, ErrNotFound
	}
	return p.SupporterViewer.ViewForSupporter(ctx, supporter)
}
