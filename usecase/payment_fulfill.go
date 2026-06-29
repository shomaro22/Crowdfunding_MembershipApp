package usecase

import (
	"context"
	"time"

	"crowdfund-app/model"
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
		purchase *model.Purchase
		err      error
	)
	purchase, err = p.Purchases.FindIntentID(ctx, intentID)
	if isRepoNotFound(err) {
		return nil
	}
	if err != nil {
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
	purchase *model.Purchase,
	planID uuid.UUID,
	period periodRange,
	customerID, subscriptionID, invoiceID *string,
) error {
	return repository.WithTx(ctx, p.DB, func(txCtx context.Context) error {
		var (
			locked     *model.Purchase
			membership *model.ProjectSupporter
			supporter  model.ProjectSupporter
			resolved  uuid.UUID
			plans     []model.ProjectPlan
			now       time.Time
			err       error
		)
		locked, err = p.Purchases.FindIDLock(txCtx, purchase.ID)
		if err != nil {
			return mapRepoNotFound(err)
		}
		purchase = locked
		if purchase.Status == model.PurchaseStatusPaid {
			return nil
		}
		if purchase.Status != model.PurchaseStatusPending {
			return nil
		}
		now = time.Now().UTC()
		membership, err = p.Supporters.FindMembership(txCtx, purchase.ProjectID, purchase.BuyerID)
		if isRepoNotFound(err) {
			membership = nil
		} else if err != nil {
			return err
		}
		if membership != nil {
			return p.markPurchasePaid(txCtx, purchase, membership.ID, period, invoiceID)
		}
		_, plans, _, err = p.Projects.FindID(txCtx, purchase.ProjectID)
		if err != nil {
			return mapRepoNotFound(err)
		}
		resolved = planID
		if resolved == uuid.Nil {
			resolved = findPlanForPurchase(plans, purchase)
		}
		if resolved == uuid.Nil {
			return ErrNotFound
		}
		supporter = model.ProjectSupporter{
			ID:                   uuid.New(),
			ProjectID:            purchase.ProjectID,
			UserID:               purchase.BuyerID,
			PlanID:               resolved,
			Status:               model.SupporterStatusActive,
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

func billingModeForPurchase(purchase *model.Purchase) model.BillingMode {
	if purchase.PaymentMethod == model.PaymentMethodKonbini {
		return model.BillingModeKonbini
	}
	return model.BillingModeAuto
}

func findPlanForPurchase(plans []model.ProjectPlan, purchase *model.Purchase) uuid.UUID {
	if len(plans) == 1 {
		return plans[0].ID
	}
	for _, plan := range plans {
		if plan.Price == purchase.TotalPrice {
			return plan.ID
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

func (p *PaymentUseCase) markPurchasePaid(ctx context.Context, purchase *model.Purchase, supporterID uuid.UUID, period periodRange, invoiceID *string) error {
	var (
		from model.PurchaseStatus
		err  error
	)
	from = purchase.Status
	purchase.Status = model.PurchaseStatusPaid
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
	return p.Purchases.RecordEvent(ctx, &model.PurchaseEvent{
		ID:         uuid.New(),
		PurchaseID: purchase.ID,
		FromStatus: &from,
		ToStatus:   model.PurchaseStatusPaid,
		CreatedAt:  time.Now().UTC(),
	})
}

func (p *PaymentUseCase) markPurchaseFailed(ctx context.Context, purchase *model.Purchase, reason string) error {
	return p.markPurchaseTerminal(ctx, purchase, model.PurchaseStatusFailed, reason)
}

func (p *PaymentUseCase) markPurchaseCancelled(ctx context.Context, purchase *model.Purchase, reason string) error {
	return p.markPurchaseTerminal(ctx, purchase, model.PurchaseStatusCancelled, reason)
}

func (p *PaymentUseCase) markPurchaseTerminal(ctx context.Context, purchase *model.Purchase, to model.PurchaseStatus, reason string) error {
	var (
		from model.PurchaseStatus
		err  error
	)
	from = purchase.Status
	purchase.Status = to
	purchase.UpdatedAt = time.Now().UTC()
	if err = p.Purchases.Update(ctx, purchase); err != nil {
		return err
	}
	if err = p.Purchases.RecordEvent(ctx, &model.PurchaseEvent{
		ID:         uuid.New(),
		PurchaseID: purchase.ID,
		FromStatus: &from,
		ToStatus:   to,
		Reason:     &reason,
		CreatedAt:  time.Now().UTC(),
	}); err != nil {
		return err
	}
	if to == model.PurchaseStatusCancelled || to == model.PurchaseStatusFailed {
		p.logPurchaseCancelled(ctx, purchase, reason, to)
	}
	return nil
}
