package usecase

import (
	"context"

	"crowdfund-app/entity"
	"crowdfund-app/repository"
	"crowdfund-app/stripe"

	"github.com/google/uuid"
	stripesdk "github.com/stripe/stripe-go/v82"
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
		SuccessURL:       checkoutURL(in.SuccessURL, p.Stripe.AppURL()+"/support/success?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:        checkoutURL(in.CancelURL, p.Stripe.AppURL()+"/support/cancel"),
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
	err = repository.WithTx(ctx, p.DB, func(txCtx context.Context) error {
		return p.Purchases.Create(txCtx, purchase)
	})
	if err != nil {
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
	purchase = p.newPendingPurchase(sc, entity.PaymentMethodCard, nil, nil)
	meta = paymentMetadata(purchase.ID, in.ProjectID, in.PlanID, in.BuyerID, entity.BillingModeAuto)
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
	err = repository.WithTx(ctx, p.DB, func(txCtx context.Context) error {
		return p.Purchases.Create(txCtx, purchase)
	})
	if err != nil {
		return IntentOut{}, err
	}
	p.logPaymentCheckoutStarted(ctx, purchase, in.Meta, "intent")
	p.logPurchaseCreated(ctx, purchase, in.Meta)
	return IntentOut{ClientSecret: sub.ClientSecret, PaymentIntentID: sub.PaymentIntentID}, nil
}

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
	if p.SupporterViewer == nil {
		return SupporterView{}, ErrNotFound
	}
	return p.SupporterViewer.ViewForSupporter(ctx, supporter)
}
