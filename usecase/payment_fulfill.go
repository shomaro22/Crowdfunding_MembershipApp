package usecase

import (
	"context"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"
	"crowdfund-app/stripe"

	"github.com/google/uuid"
	stripesdk "github.com/stripe/stripe-go/v82"
)

func (p *PaymentUseCase) fulfillCheckoutSession(ctx context.Context, sess stripesdk.CheckoutSession) error {
	var (
		purchase *entity.Purchase
		planID   uuid.UUID
		err      error
	)
	purchase, err = p.Purchases.FindBySessionID(ctx, sess.ID)
	if err != nil || purchase == nil {
		return err
	}
	planID, err = uuid.Parse(sess.Metadata[stripe.MetaPlanID])
	if err != nil {
		planID = uuid.Nil
	}
	return p.fulfillPurchase(ctx, purchase, planID, defaultPeriod(nil, nil), nil, nil, nil)
}

func (p *PaymentUseCase) fulfillPaymentIntent(ctx context.Context, pi stripesdk.PaymentIntent) error {
	var (
		planID uuid.UUID
		err    error
	)
	planID, err = uuid.Parse(pi.Metadata[stripe.MetaPlanID])
	if err != nil {
		planID = uuid.Nil
	}
	return p.fulfillPaymentIntentID(ctx, pi.ID, time.Time{}, time.Time{}, nil, nil, nil, planID)
}

func (p *PaymentUseCase) fulfillPaymentIntentID(
	ctx context.Context,
	intentID string,
	start, end time.Time,
	customerID, subscriptionID, invoiceID *string,
	planID uuid.UUID,
) error {
	var (
		purchase *entity.Purchase
		err      error
	)
	purchase, err = p.Purchases.FindByIntentID(ctx, intentID)
	if err != nil || purchase == nil {
		return err
	}
	period := defaultPeriod(nil, nil)
	if !start.IsZero() && !end.IsZero() {
		period = periodRange{start, end}
	}
	return p.fulfillPurchase(ctx, purchase, planID, period, customerID, subscriptionID, invoiceID)
}

func (p *PaymentUseCase) fulfillPurchaseID(ctx context.Context, purchaseID uuid.UUID, start, end time.Time, customerID, subscriptionID, invoiceID *string) error {
	var (
		purchase *entity.Purchase
		err      error
	)
	purchase, err = p.Purchases.FindByID(ctx, purchaseID)
	if err != nil || purchase == nil {
		return err
	}
	return p.fulfillPurchase(ctx, purchase, uuid.Nil, periodRange{start, end}, customerID, subscriptionID, invoiceID)
}

func defaultPeriod(start, end *time.Time) periodRange {
	var now time.Time
	now = time.Now().UTC()
	if start != nil && end != nil {
		return periodRange{*start, *end}
	}
	return periodRange{now, now.AddDate(0, 0, supportPeriodDays)}
}

func (p *PaymentUseCase) fulfillPurchase(
	ctx context.Context,
	purchase *entity.Purchase,
	planID uuid.UUID,
	period periodRange,
	customerID, subscriptionID, invoiceID *string,
) error {
	if purchase.Status == entity.PurchaseStatusPaid {
		return nil
	}
	if purchase.Status != entity.PurchaseStatusPending {
		return nil
	}
	return repository.WithTx(ctx, p.DB, func(txCtx context.Context) error {
		var (
			active    *entity.ProjectSupporter
			supporter entity.ProjectSupporter
			resolved  uuid.UUID
			proj      *entity.Project
			plans     []entity.ProjectPlan
			now       time.Time
			err       error
		)
		now = time.Now().UTC()
		active, err = p.Supporters.FindActive(txCtx, purchase.ProjectID, purchase.BuyerID)
		if err != nil {
			return err
		}
		if active != nil {
			return p.markPurchasePaid(txCtx, purchase, active.ID, period, invoiceID)
		}
		proj, plans, _, err = p.Projects.FindByID(txCtx, purchase.ProjectID)
		if err != nil || proj == nil {
			return ErrNotFound
		}
		resolved = planID
		if resolved == uuid.Nil {
			resolved = findPlanForPurchase(plans, purchase)
		}
		if resolved == uuid.Nil {
			return ErrNotFound
		}
		supporter = entity.ProjectSupporter{
			ID:                   uuid.New(),
			ProjectID:            purchase.ProjectID,
			UserID:               purchase.BuyerID,
			PlanID:               resolved,
			Status:               entity.SupporterStatusActive,
			BillingMode:          billingModeForPurchase(purchase),
			CurrentPeriodStart:   period.start,
			CurrentPeriodEnd:     period.end,
			StripeCustomerID:     customerID,
			StripeSubscriptionID: subscriptionID,
			CreatedAt:            now,
			UpdatedAt:            now,
		}
		if err = p.Supporters.Create(txCtx, &supporter); err != nil {
			return err
		}
		if err = p.Projects.IncrementSupportCounts(txCtx, purchase.ProjectID, resolved); err != nil {
			return err
		}
		return p.markPurchasePaid(txCtx, purchase, supporter.ID, period, invoiceID)
	})
}

func billingModeForPurchase(purchase *entity.Purchase) entity.BillingMode {
	if purchase.PaymentMethod == entity.PaymentMethodKonbini {
		return entity.BillingModeKonbini
	}
	return entity.BillingModeAuto
}

func findPlanForPurchase(plans []entity.ProjectPlan, purchase *entity.Purchase) uuid.UUID {
	if len(plans) == 1 {
		return plans[0].ID
	}
	for _, pl := range plans {
		if pl.Price == purchase.TotalPrice {
			return pl.ID
		}
	}
	if len(plans) > 0 {
		return plans[0].ID
	}
	return uuid.Nil
}

func stripeString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (p *PaymentUseCase) markPurchasePaid(ctx context.Context, purchase *entity.Purchase, supporterID uuid.UUID, period periodRange, invoiceID *string) error {
	var (
		from entity.PurchaseStatus
		err  error
	)
	from = purchase.Status
	purchase.Status = entity.PurchaseStatusPaid
	purchase.SupporterID = &supporterID
	purchase.PeriodStart = &period.start
	purchase.PeriodEnd = &period.end
	purchase.StripeInvoiceID = invoiceID
	purchase.UpdatedAt = time.Now().UTC()
	if err = p.Purchases.Update(ctx, purchase); err != nil {
		return err
	}
	return p.Purchases.RecordEvent(ctx, &entity.PurchaseEvent{
		ID:         uuid.New(),
		PurchaseID: purchase.ID,
		FromStatus: &from,
		ToStatus:   entity.PurchaseStatusPaid,
		CreatedAt:  time.Now().UTC(),
	})
}

func (p *PaymentUseCase) markPurchaseFailed(ctx context.Context, purchase *entity.Purchase, reason string) error {
	var (
		from entity.PurchaseStatus
		err  error
	)
	from = purchase.Status
	purchase.Status = entity.PurchaseStatusFailed
	purchase.UpdatedAt = time.Now().UTC()
	if err = p.Purchases.Update(ctx, purchase); err != nil {
		return err
	}
	return p.Purchases.RecordEvent(ctx, &entity.PurchaseEvent{
		ID:         uuid.New(),
		PurchaseID: purchase.ID,
		FromStatus: &from,
		ToStatus:   entity.PurchaseStatusFailed,
		Reason:     &reason,
		CreatedAt:  time.Now().UTC(),
	})
}
