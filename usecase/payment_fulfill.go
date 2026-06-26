package usecase

import (
	"context"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"

	"github.com/google/uuid"
)

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
	purchase, err = p.Purchases.FindIntentID(ctx, intentID)
	if err != nil || purchase == nil {
		return err
	}
	period := defaultPeriod(nil, nil)
	if !start.IsZero() && !end.IsZero() {
		period = periodRange{start, end}
	}
	return p.fulfillFirstSupport(ctx, purchase, planID, period, customerID, subscriptionID, invoiceID)
}

func defaultPeriod(start, end *time.Time) periodRange {
	var now time.Time
	now = time.Now().UTC()
	if start != nil && end != nil {
		return periodRange{*start, *end}
	}
	return periodRange{now, now.AddDate(0, 0, supportPeriodDays)}
}

func (p *PaymentUseCase) fulfillFirstSupport(
	ctx context.Context,
	purchase *entity.Purchase,
	planID uuid.UUID,
	period periodRange,
	customerID, subscriptionID, invoiceID *string,
) error {
	return repository.WithTx(ctx, p.DB, func(txCtx context.Context) error {
		var (
			locked     *entity.Purchase
			membership *entity.ProjectSupporter
			supporter  entity.ProjectSupporter
			resolved  uuid.UUID
			proj      *entity.Project
			plans     []entity.ProjectPlan
			now       time.Time
			err       error
		)
		locked, err = p.Purchases.FindIDLock(txCtx, purchase.ID)
		if err != nil || locked == nil {
			return ErrNotFound
		}
		purchase = locked
		if purchase.Status == entity.PurchaseStatusPaid {
			return nil
		}
		if purchase.Status != entity.PurchaseStatusPending {
			return nil
		}
		now = time.Now().UTC()
		membership, err = p.Supporters.FindMembership(txCtx, purchase.ProjectID, purchase.BuyerID)
		if err != nil {
			return err
		}
		if membership != nil {
			return p.markPurchasePaid(txCtx, purchase, membership.ID, period, invoiceID)
		}
		proj, plans, _, err = p.Projects.FindID(txCtx, purchase.ProjectID)
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
	if err = p.Projects.AddCurrentAmount(ctx, purchase.ProjectID, int64(purchase.TotalPrice)); err != nil {
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
	return p.markPurchaseTerminal(ctx, purchase, entity.PurchaseStatusFailed, reason)
}

func (p *PaymentUseCase) markPurchaseCancelled(ctx context.Context, purchase *entity.Purchase, reason string) error {
	return p.markPurchaseTerminal(ctx, purchase, entity.PurchaseStatusCancelled, reason)
}

func (p *PaymentUseCase) markPurchaseTerminal(ctx context.Context, purchase *entity.Purchase, to entity.PurchaseStatus, reason string) error {
	var (
		from entity.PurchaseStatus
		err  error
	)
	from = purchase.Status
	purchase.Status = to
	purchase.UpdatedAt = time.Now().UTC()
	if err = p.Purchases.Update(ctx, purchase); err != nil {
		return err
	}
	return p.Purchases.RecordEvent(ctx, &entity.PurchaseEvent{
		ID:         uuid.New(),
		PurchaseID: purchase.ID,
		FromStatus: &from,
		ToStatus:   to,
		Reason:     &reason,
		CreatedAt:  time.Now().UTC(),
	})
}
