package usecase

import (
	"context"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
)

func (s *SupporterUseCase) Cancel(ctx context.Context, userID, supporterID uuid.UUID) (SupporterView, error) {
	var (
		supporter *entity.ProjectSupporter
		now       time.Time
		err       error
	)
	now = time.Now().UTC()
	supporter, err = s.loadOwned(ctx, userID, supporterID)
	if err != nil {
		return SupporterView{}, err
	}
	if !supporterCanManage(supporter, now) || supporter.Status == entity.SupporterStatusCancelled {
		return SupporterView{}, ErrForbidden
	}
	if supporter.BillingMode == entity.BillingModeAuto &&
		supporter.StripeSubscriptionID != nil &&
		s.Stripe != nil {
		if err = s.Stripe.CancelSubscriptionAtPeriodEnd(ctx, *supporter.StripeSubscriptionID); err != nil {
			return SupporterView{}, err
		}
	}
	supporter.Status = entity.SupporterStatusCancelled
	supporter.UpdatedAt = now
	if err = s.Supporters.Update(ctx, supporter); err != nil {
		return SupporterView{}, err
	}
	return s.enrichView(ctx, supporter)
}

func (s *SupporterUseCase) Rejoin(ctx context.Context, userID, supporterID uuid.UUID) (SupporterView, error) {
	var (
		supporter *entity.ProjectSupporter
		now       time.Time
		err       error
	)
	now = time.Now().UTC()
	supporter, err = s.loadOwned(ctx, userID, supporterID)
	if err != nil {
		return SupporterView{}, err
	}
	if supporter.Status != entity.SupporterStatusCancelled {
		return SupporterView{}, ErrBadRequest
	}
	if now.After(supporter.CurrentPeriodEnd) {
		return SupporterView{}, ErrBadRequest
	}
	if supporter.BillingMode == entity.BillingModeAuto &&
		supporter.StripeSubscriptionID != nil &&
		s.Stripe != nil {
		if err = s.Stripe.ResumeSubscription(ctx, *supporter.StripeSubscriptionID); err != nil {
			return SupporterView{}, err
		}
	}
	supporter.Status = entity.SupporterStatusActive
	supporter.UpdatedAt = now
	if err = s.Supporters.Update(ctx, supporter); err != nil {
		return SupporterView{}, err
	}
	return s.enrichView(ctx, supporter)
}

type ChangeBillingIn struct {
	BillingMode   entity.BillingMode
	PaymentMethod entity.PaymentMethod
}

func (s *SupporterUseCase) ChangeBilling(ctx context.Context, userID, supporterID uuid.UUID, in ChangeBillingIn) (SupporterView, error) {
	var (
		supporter *entity.ProjectSupporter
		now       time.Time
		err       error
	)
	if !in.BillingMode.Valid() {
		return SupporterView{}, ErrBadRequest
	}
	now = time.Now().UTC()
	supporter, err = s.loadOwned(ctx, userID, supporterID)
	if err != nil {
		return SupporterView{}, err
	}
	if !supporterCanManage(supporter, now) || supporter.Status == entity.SupporterStatusCancelled {
		return SupporterView{}, ErrForbidden
	}
	if supporter.PendingBillingMode != nil {
		return SupporterView{}, ErrBadRequest
	}
	if in.BillingMode == supporter.BillingMode {
		if in.BillingMode != entity.BillingModeAuto {
			return SupporterView{}, ErrBadRequest
		}
		if in.PaymentMethod == "" {
			return SupporterView{}, ErrBadRequest
		}
		return SupporterView{}, ErrBadRequest
	}
	changeAt := supporter.CurrentPeriodEnd
	mode := in.BillingMode
	supporter.PendingBillingMode = &mode
	supporter.PlanChangeDate = &changeAt
	supporter.UpdatedAt = now
	if in.BillingMode == entity.BillingModeKonbini &&
		supporter.BillingMode == entity.BillingModeAuto &&
		supporter.StripeSubscriptionID != nil &&
		s.Stripe != nil {
		if err = s.Stripe.CancelSubscriptionAtPeriodEnd(ctx, *supporter.StripeSubscriptionID); err != nil {
			return SupporterView{}, err
		}
	}
	if err = s.Supporters.Update(ctx, supporter); err != nil {
		return SupporterView{}, err
	}
	return s.enrichView(ctx, supporter)
}

func (s *SupporterUseCase) applyBillingModeSwitch(ctx context.Context, supporter *entity.ProjectSupporter, mode entity.BillingMode) error {
	if mode == entity.BillingModeKonbini {
		supporter.StripeSubscriptionID = nil
		return nil
	}
	_ = ctx
	return nil
}
