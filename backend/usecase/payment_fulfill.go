package usecase

import (
	"context"
	"time"

	"crowdfund-app/events"
	"crowdfund-app/model"
	"crowdfund-app/stripe"

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
		purchase model.Purchase
		err      error
	)
	purchase, err = p.Purchases.GetIntentID(ctx, intentID)
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

func (p *PaymentUseCase) fulfillFirstSupport(
	ctx context.Context,
	purchase model.Purchase,
	planID uuid.UUID,
	period periodRange,
	customerID, subscriptionID, invoiceID *string,
) error {
	var (
		completed *events.SupportCompleted
		err       error
	)

	err = p.Tx.WithTx(ctx, func(txCtx context.Context) error {
		var (
			membership model.ProjectSupporter
			hasMember  bool
			supporter  model.ProjectSupporter
			resolved   uuid.UUID
			plans      []model.ProjectPlan
			now        time.Time
			txErr      error
		)
		purchase, txErr = p.Purchases.GetIDLock(txCtx, purchase.ID)
		if txErr != nil {
			return mapRepoNotFound(txErr)
		}
		if purchase.Status == model.PurchaseStatusPaid {
			return nil
		}
		if purchase.Status != model.PurchaseStatusPending {
			return nil
		}
		now = time.Now().UTC()
		membership, txErr = p.Supporters.GetMembership(txCtx, purchase.ProjectID, purchase.BuyerID)
		if isRepoNotFound(txErr) {
			hasMember = false
		} else if txErr != nil {
			return txErr
		} else {
			hasMember = true
		}
		if hasMember {
			completed = &events.SupportCompleted{
				PurchaseID:  purchase.ID,
				ProjectID:   purchase.ProjectID,
				BuyerID:     purchase.BuyerID,
				SupporterID: membership.ID,
				PlanID:      membership.PlanID,
				Amount:      int64(purchase.TotalPrice),
				PeriodStart: period.start,
				PeriodEnd:   period.end,
			}
			return p.markPurchasePaid(txCtx, purchase, membership.ID, period, invoiceID)
		}
		_, plans, _, txErr = p.Projects.GetID(txCtx, purchase.ProjectID)
		if txErr != nil {
			return mapRepoNotFound(txErr)
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
		if txErr = p.Supporters.Create(txCtx, &supporter); txErr != nil {
			return txErr
		}
		if txErr = p.Projects.IncrementSupportCounts(txCtx, purchase.ProjectID, resolved); txErr != nil {
			return txErr
		}
		completed = &events.SupportCompleted{
			PurchaseID:  purchase.ID,
			ProjectID:   purchase.ProjectID,
			BuyerID:     purchase.BuyerID,
			SupporterID: supporter.ID,
			PlanID:      resolved,
			Amount:      int64(purchase.TotalPrice),
			PeriodStart: period.start,
			PeriodEnd:   period.end,
		}
		return p.markPurchasePaid(txCtx, purchase, supporter.ID, period, invoiceID)
	})
	if err != nil {
		return err
	}
	if completed != nil && p.supportEvents != nil {
		return p.supportEvents.Publish(ctx, *completed)
	}
	return nil
}

func (p *PaymentUseCase) markPurchasePaid(ctx context.Context, purchase model.Purchase, supporterID uuid.UUID, period periodRange, invoiceID *string) error {
	var (
		supID = supporterID
		start = period.start
		end   = period.end
	)
	return MarkPurchasePaid(ctx, p.Purchases, p.Projects, purchase, MarkPaidOptions{
		SupporterID: &supID,
		PeriodStart: &start,
		PeriodEnd:   &end,
		InvoiceID:   invoiceID,
		AddFunding:  true,
	})
}

func (p *PaymentUseCase) markPurchaseFailed(ctx context.Context, purchase model.Purchase, reason string) error {
	return p.markPurchaseTerminal(ctx, purchase, model.PurchaseStatusFailed, reason)
}

func (p *PaymentUseCase) markPurchaseCancelled(ctx context.Context, purchase model.Purchase, reason string) error {
	return p.markPurchaseTerminal(ctx, purchase, model.PurchaseStatusCancelled, reason)
}

func (p *PaymentUseCase) markPurchaseTerminal(ctx context.Context, purchase model.Purchase, to model.PurchaseStatus, reason string) error {
	var (
		from model.PurchaseStatus
		err  error
	)
	from = purchase.Status
	purchase.Status = to
	purchase.UpdatedAt = time.Now().UTC()
	if err = p.Purchases.Update(ctx, &purchase); err != nil {
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

func (p *PaymentUseCase) fulfillCheckoutSession(ctx context.Context, sess stripe.CheckoutSessionSnapshot) error {
	var (
		purchase model.Purchase
		planID   uuid.UUID
		purpose  checkoutPurpose
		err      error
	)
	purchase, err = p.Purchases.GetSessionID(ctx, sess.ID)
	if isRepoNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if purchase.Status == model.PurchaseStatusPaid {
		return nil
	}
	if purchase.Status != model.PurchaseStatusPending {
		return nil
	}
	planID, err = uuid.Parse(sess.Metadata[stripe.MetaPlanID])
	if err != nil {
		planID = uuid.Nil
	}
	purpose = parseCheckoutPurpose(sess.Metadata)
	if purpose == checkoutKonbiniRenewal {
		return p.fulfillKonbiniRenewal(ctx, purchase, planID, time.Now().UTC())
	}
	if isKonbiniPurchase(purchase) {
		return p.fulfillKonbiniFirstSupport(ctx, purchase, planID, time.Now().UTC())
	}
	return p.fulfillFirstSupport(ctx, purchase, planID, defaultPeriod(nil, nil), nil, nil, nil)
}

func (p *PaymentUseCase) fulfillKonbiniFirstSupport(ctx context.Context, purchase model.Purchase, planID uuid.UUID, paidAt time.Time) error {
	return p.fulfillFirstSupport(ctx, purchase, planID, konbiniFirstPeriod(paidAt), nil, nil, nil)
}

func (p *PaymentUseCase) fulfillKonbiniRenewal(ctx context.Context, purchase model.Purchase, planID uuid.UUID, paidAt time.Time) error {
	return p.Tx.WithTx(ctx, func(txCtx context.Context) error {
		var (
			supporter model.ProjectSupporter
			period    periodRange
			now       time.Time
			err       error
		)
		purchase, err = p.Purchases.GetIDLock(txCtx, purchase.ID)
		if err != nil {
			return mapRepoNotFound(err)
		}
		if purchase.Status == model.PurchaseStatusPaid {
			return nil
		}
		if purchase.Status != model.PurchaseStatusPending {
			return nil
		}
		if purchase.SupporterID == nil {
			return ErrNotFound
		}
		supporter, err = p.Supporters.GetID(txCtx, *purchase.SupporterID)
		if err != nil {
			return mapRepoNotFound(err)
		}
		period = konbiniRenewalPeriod(supporter, paidAt)
		now = time.Now().UTC()
		supporter.Status = model.SupporterStatusActive
		supporter.CurrentPeriodStart = period.start
		supporter.CurrentPeriodEnd = period.end
		if planID != uuid.Nil {
			supporter.PlanID = planID
		}
		supporter.UpdatedAt = now
		if err = p.Supporters.Update(txCtx, &supporter); err != nil {
			return err
		}
		return p.markPurchasePaid(txCtx, purchase, supporter.ID, period, nil)
	})
}

func (p *PaymentUseCase) handleKonbiniCheckoutUnpaid(ctx context.Context, purchase model.Purchase, reason string, cancelled bool) error {
	if purchase.Status != model.PurchaseStatusPending {
		return nil
	}
	if !isKonbiniPurchase(purchase) {
		if cancelled {
			return p.markPurchaseCancelled(ctx, purchase, reason)
		}
		return p.markPurchaseFailed(ctx, purchase, reason)
	}
	if purchase.SupporterID != nil {
		return p.failKonbiniRenewal(ctx, purchase, reason, cancelled)
	}
	if cancelled {
		return p.markPurchaseCancelled(ctx, purchase, reason)
	}
	return p.markPurchaseFailed(ctx, purchase, reason)
}

func (p *PaymentUseCase) failKonbiniRenewal(ctx context.Context, purchase model.Purchase, reason string, cancelled bool) error {
	return p.Tx.WithTx(ctx, func(txCtx context.Context) error {
		var (
			supporter model.ProjectSupporter
			err       error
		)
		if cancelled {
			err = p.markPurchaseCancelled(txCtx, purchase, reason)
		} else {
			err = p.markPurchaseFailed(txCtx, purchase, reason)
		}
		if err != nil {
			return err
		}
		if purchase.SupporterID == nil {
			return nil
		}
		supporter, err = p.Supporters.GetID(txCtx, *purchase.SupporterID)
		if isRepoNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if supporter.Status == model.SupporterStatusPendingRenewal {
			return nil
		}
		supporter.Status = model.SupporterStatusPendingRenewal
		supporter.UpdatedAt = time.Now().UTC()
		return p.Supporters.Update(txCtx, &supporter)
	})
}

func (p *PaymentUseCase) fulfillPaymentIntent(ctx context.Context, pi stripe.PaymentIntentDetail) error {
	var (
		planID uuid.UUID
		err    error
	)
	if pi.Metadata[stripe.MetaCheckoutPurpose] == stripe.PurposePlanUpgrade {
		return nil
	}
	if pi.Metadata[stripe.MetaBillingMode] == string(model.BillingModeAuto) {
		planID, err = uuid.Parse(pi.Metadata[stripe.MetaPlanID])
		if err != nil {
			planID = uuid.Nil
		}
		return p.fulfillPaymentIntentID(ctx, pi.ID, time.Time{}, time.Time{}, nil, nil, nil, planID)
	}
	if pi.Metadata[stripe.MetaBillingMode] != string(model.BillingModeKonbini) {
		return nil
	}
	return p.fulfillKonbiniPaymentIntent(ctx, pi)
}

func (p *PaymentUseCase) fulfillKonbiniPaymentIntent(ctx context.Context, pi stripe.PaymentIntentDetail) error {
	var (
		purchase model.Purchase
		planID   uuid.UUID
		purpose  checkoutPurpose
		err      error
	)
	purchase, err = p.Purchases.GetIntentID(ctx, pi.ID)
	if isRepoNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	planID, err = uuid.Parse(pi.Metadata[stripe.MetaPlanID])
	if err != nil {
		planID = uuid.Nil
	}
	purpose = parseCheckoutPurpose(pi.Metadata)
	if purpose == checkoutKonbiniRenewal {
		return p.fulfillKonbiniRenewal(ctx, purchase, planID, time.Now().UTC())
	}
	return p.fulfillKonbiniFirstSupport(ctx, purchase, planID, time.Now().UTC())
}
