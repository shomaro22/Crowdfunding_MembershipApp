package usecase

import (
	"context"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"
	"crowdfund-app/stripe"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UpgradeOut struct {
	Supporter                 SupporterView
	PaymentIntentClientSecret *string
}

type SupporterUseCase struct {
	DB         *gorm.DB
	Users      repository.UserRepo
	Projects   repository.ProjectRepo
	Supporters repository.SupporterRepo
	Purchases  repository.PurchaseRepo
	Connect    repository.ConnectRepo
	Activity   repository.ActivityRepo
	Stripe     stripe.Client
}

type ChangeBillingIn struct {
	BillingMode   string
	PaymentMethod string
}

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

func (s *SupporterUseCase) ChangeBilling(ctx context.Context, userID, supporterID uuid.UUID, in ChangeBillingIn) (SupporterView, error) {
	var (
		supporter *entity.ProjectSupporter
		mode      entity.BillingMode
		now       time.Time
		err       error
	)
	mode = entity.BillingMode(in.BillingMode)
	if !mode.Valid() {
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
	if mode == supporter.BillingMode {
		return SupporterView{}, ErrBadRequest
	}
	changeAt := supporter.CurrentPeriodEnd
	supporter.PendingBillingMode = &mode
	supporter.PlanChangeDate = &changeAt
	supporter.UpdatedAt = now
	if mode == entity.BillingModeKonbini &&
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

func (s *SupporterUseCase) applyBillingMode(ctx context.Context, supporter *entity.ProjectSupporter, mode entity.BillingMode) error {
	switch mode {
	case entity.BillingModeKonbini:
		supporter.StripeSubscriptionID = nil
		return nil
	case entity.BillingModeAuto:
		if supporter.StripeSubscriptionID == nil {
			return ErrBadRequest
		}
		if s.Stripe == nil {
			return ErrStripeUnavailable
		}
		return s.Stripe.ResumeSubscription(ctx, *supporter.StripeSubscriptionID)
	default:
		return ErrBadRequest
	}
}

func supporterCanManage(s *entity.ProjectSupporter, now time.Time) bool {
	if s == nil {
		return false
	}
	switch s.Status {
	case entity.SupporterStatusActive, entity.SupporterStatusPendingRenewal, entity.SupporterStatusCancelled:
		return s.HasBenefits(now)
	default:
		return false
	}
}
