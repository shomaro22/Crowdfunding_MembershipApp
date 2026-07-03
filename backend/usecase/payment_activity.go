package usecase

import (
	"context"

	"crowdfund-app/model"

	"github.com/google/uuid"
)

func (p *PaymentUseCase) logPurchaseCreated(ctx context.Context, purchase model.Purchase, meta RequestMeta) {
	uid := &purchase.BuyerID
	LogActivity(ctx, p.Activity, uid, model.PurchaseCreated, "purchase", purchase.ID.String(), meta, purchaseActivityMeta(purchase))
}

func (p *PaymentUseCase) logPaymentCheckoutStarted(ctx context.Context, purchase model.Purchase, meta RequestMeta, flow string) {
	uid := &purchase.BuyerID
	fields := map[string]string{
		"flow":       flow,
		"project_id": purchase.ProjectID.String(),
	}
	LogActivity(ctx, p.Activity, uid, model.PaymentCheckout, "purchase", purchase.ID.String(), meta, activityMeta(fields))
}

func (p *PaymentUseCase) logPurchaseCancelled(ctx context.Context, purchase model.Purchase, reason string, to model.PurchaseStatus) {
	uid := &purchase.BuyerID
	LogActivity(ctx, p.Activity, uid, model.PurchaseCancelled, "purchase", purchase.ID.String(), RequestMeta{}, activityMeta(map[string]string{
		"reason":    reason,
		"to_status": string(to),
	}))
}

func (p *PaymentUseCase) logPaymentWebhook(ctx context.Context, eventType, eventID string) {
	LogActivity(ctx, p.Activity, nil, model.PaymentWebhook, "stripe_event", eventID, RequestMeta{}, activityMeta(map[string]string{
		"event_type": eventType,
	}))
}

func (p *PaymentUseCase) newPendingPurchase(
	sc supportContext,
	pm model.PaymentMethod,
	checkoutID, intentID *string,
) model.Purchase {
	var (
		row model.Purchase
		key string
	)
	key = uuid.New().String()
	row = model.Purchase{
		ID:               uuid.New(),
		ProjectID:        sc.project.ID,
		BuyerID:          sc.buyer.ID,
		Quantity:         1,
		TotalPrice:       sc.plan.Price,
		Status:           model.PurchaseStatusPending,
		PaymentMethod:    pm,
		StripeCheckoutID: checkoutID,
		StripeIntentID:   intentID,
		IdempotencyKey:   &key,
		CreatedAt:        sc.now,
		UpdatedAt:        sc.now,
	}
	if sc.purpose == checkoutKonbiniRenewal && sc.hasMembership {
		row.SupporterID = &sc.membership.ID
	}
	return row
}

func (p *PaymentUseCase) persistPendingPurchase(ctx context.Context, purchase model.Purchase) error {
	return p.Tx.WithTx(ctx, func(txCtx context.Context) error {
		return p.Purchases.Create(txCtx, &purchase)
	})
}
