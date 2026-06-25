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
		purpose  checkoutPurpose
		err      error
	)
	purchase, err = p.Purchases.FindBySessionID(ctx, sess.ID)
	if err != nil || purchase == nil {
		return err
	}
	if purchase.Status == entity.PurchaseStatusPaid {
		return nil
	}
	if purchase.Status != entity.PurchaseStatusPending {
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

func (p *PaymentUseCase) fulfillKonbiniFirstSupport(ctx context.Context, purchase *entity.Purchase, planID uuid.UUID, paidAt time.Time) error {
	return p.fulfillFirstSupport(ctx, purchase, planID, konbiniFirstPeriod(paidAt), nil, nil, nil)
}

func (p *PaymentUseCase) fulfillKonbiniRenewal(ctx context.Context, purchase *entity.Purchase, planID uuid.UUID, paidAt time.Time) error {
	var (
		supporter *entity.ProjectSupporter
		period    periodRange
		err       error
	)
	if purchase.Status == entity.PurchaseStatusPaid {
		return nil
	}
	if purchase.Status != entity.PurchaseStatusPending {
		return nil
	}
	if purchase.SupporterID == nil {
		return ErrNotFound
	}
	supporter, err = p.Supporters.FindByID(ctx, *purchase.SupporterID)
	if err != nil || supporter == nil {
		return ErrNotFound
	}
	period = konbiniRenewalPeriod(supporter, paidAt)
	return repository.WithTx(ctx, p.DB, func(txCtx context.Context) error {
		var now time.Time
		now = time.Now().UTC()
		supporter.Status = entity.SupporterStatusActive
		supporter.CurrentPeriodStart = period.start
		supporter.CurrentPeriodEnd = period.end
		if planID != uuid.Nil {
			supporter.PlanID = planID
		}
		supporter.UpdatedAt = now
		if err = p.Supporters.Update(txCtx, supporter); err != nil {
			return err
		}
		return p.markPurchasePaid(txCtx, purchase, supporter.ID, period, nil)
	})
}

func (p *PaymentUseCase) handleKonbiniCheckoutUnpaid(ctx context.Context, purchase *entity.Purchase, reason string, cancelled bool) error {
	if purchase == nil || purchase.Status != entity.PurchaseStatusPending {
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

func (p *PaymentUseCase) failKonbiniRenewal(ctx context.Context, purchase *entity.Purchase, reason string, cancelled bool) error {
	var (
		supporter *entity.ProjectSupporter
		err       error
	)
	if cancelled {
		err = p.markPurchaseCancelled(ctx, purchase, reason)
	} else {
		err = p.markPurchaseFailed(ctx, purchase, reason)
	}
	if err != nil {
		return err
	}
	supporter, err = p.Supporters.FindByID(ctx, *purchase.SupporterID)
	if err != nil || supporter == nil {
		return err
	}
	if supporter.Status == entity.SupporterStatusPendingRenewal {
		return nil
	}
	supporter.Status = entity.SupporterStatusPendingRenewal
	supporter.UpdatedAt = time.Now().UTC()
	return p.Supporters.Update(ctx, supporter)
}
func konbiniFirstPeriod(paidAt time.Time) periodRange {
	return periodRange{start: paidAt, end: paidAt.AddDate(0, 0, supportPeriodDays)}
}

func konbiniRenewalPeriod(supporter *entity.ProjectSupporter, paidAt time.Time) periodRange {
	var start time.Time
	start = supporter.CurrentPeriodEnd
	if paidAt.After(start) {
		start = paidAt
	}
	return periodRange{start: start, end: start.AddDate(0, 0, supportPeriodDays)}
}

func (p *PaymentUseCase) fulfillPaymentIntent(ctx context.Context, pi stripesdk.PaymentIntent) error {
	var (
		planID uuid.UUID
		err    error
	)
	if pi.Metadata[stripe.MetaBillingMode] == string(entity.BillingModeAuto) {
		planID, err = uuid.Parse(pi.Metadata[stripe.MetaPlanID])
		if err != nil {
			planID = uuid.Nil
		}
		return p.fulfillByPaymentIntentID(ctx, pi.ID, time.Time{}, time.Time{}, nil, nil, nil, planID)
	}
	if pi.Metadata[stripe.MetaBillingMode] != string(entity.BillingModeKonbini) {
		return nil
	}
	return p.fulfillKonbiniPaymentIntent(ctx, pi)
}

func (p *PaymentUseCase) fulfillKonbiniPaymentIntent(ctx context.Context, pi stripesdk.PaymentIntent) error {
	var (
		purchase *entity.Purchase
		planID   uuid.UUID
		purpose  checkoutPurpose
		err      error
	)
	purchase, err = p.Purchases.FindByIntentID(ctx, pi.ID)
	if err != nil || purchase == nil {
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
