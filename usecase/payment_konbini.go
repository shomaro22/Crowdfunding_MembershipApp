package usecase

import (
	"context"
	"time"

	"crowdfund-app/model"
	"crowdfund-app/repository"
	"crowdfund-app/stripe"

	"github.com/google/uuid"
)

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
	return repository.WithTx(ctx, p.DB, func(txCtx context.Context) error {
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
	return repository.WithTx(ctx, p.DB, func(txCtx context.Context) error {
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
func konbiniFirstPeriod(paidAt time.Time) periodRange {
	return periodRange{start: paidAt, end: paidAt.AddDate(0, 0, supportPeriodDays)}
}

func konbiniRenewalPeriod(supporter model.ProjectSupporter, paidAt time.Time) periodRange {
	var start time.Time
	start = supporter.CurrentPeriodEnd
	if paidAt.After(start) {
		start = paidAt
	}
	return periodRange{start: start, end: start.AddDate(0, 0, supportPeriodDays)}
}

func (p *PaymentUseCase) fulfillPaymentIntent(ctx context.Context, pi stripe.PaymentIntentDetail) error {
	var (
		planID uuid.UUID
		err    error
	)
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
