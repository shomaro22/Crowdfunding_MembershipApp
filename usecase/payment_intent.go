package usecase

import (
	"context"

	"crowdfund-app/entity"
	"crowdfund-app/repository"
	"crowdfund-app/stripe"
)

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
