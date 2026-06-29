package usecase

import (
	"context"
	"time"

	"crowdfund-app/model"
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
	Users      repository.IUserRepo
	Projects   repository.IProjectRepo
	Supporters repository.ISupporterRepo
	Purchases  repository.IPurchaseRepo
	Connect    repository.IConnectRepo
	Activity   repository.IActivityRepo
	Stripe     stripe.Client
}

type ChangeBillingIn struct {
	BillingMode   string
	PaymentMethod string
}

func (s *SupporterUseCase) Cancel(ctx context.Context, userID, supporterID uuid.UUID) (SupporterView, error) {
	var (
		supporter model.ProjectSupporter
		now       time.Time
		err       error
	)
	now = time.Now().UTC()
	supporter, err = s.loadOwned(ctx, userID, supporterID)
	if err != nil {
		return SupporterView{}, err
	}
	if !supporterCanManage(supporter, now) || supporter.Status == model.SupporterStatusCancelled {
		return SupporterView{}, ErrForbidden
	}
	if supporter.BillingMode == model.BillingModeAuto &&
		supporter.StripeSubscriptionID != nil &&
		s.Stripe != nil {
		if err = s.Stripe.CancelSubscriptionAtPeriodEnd(ctx, *supporter.StripeSubscriptionID); err != nil {
			return SupporterView{}, err
		}
	}
	supporter.Status = model.SupporterStatusCancelled
	supporter.UpdatedAt = now
	if err = s.Supporters.Update(ctx, &supporter); err != nil {
		return SupporterView{}, err
	}
	return s.enrichView(ctx, supporter)
}

func (s *SupporterUseCase) Rejoin(ctx context.Context, userID, supporterID uuid.UUID) (SupporterView, error) {
	var (
		supporter model.ProjectSupporter
		now       time.Time
		err       error
	)
	now = time.Now().UTC()
	supporter, err = s.loadOwned(ctx, userID, supporterID)
	if err != nil {
		return SupporterView{}, err
	}
	if supporter.Status != model.SupporterStatusCancelled {
		return SupporterView{}, ErrBadRequest
	}
	if now.After(supporter.CurrentPeriodEnd) {
		return SupporterView{}, ErrBadRequest
	}
	if supporter.BillingMode == model.BillingModeAuto &&
		supporter.StripeSubscriptionID != nil &&
		s.Stripe != nil {
		if err = s.Stripe.ResumeSubscription(ctx, *supporter.StripeSubscriptionID); err != nil {
			return SupporterView{}, err
		}
	}
	supporter.Status = model.SupporterStatusActive
	supporter.UpdatedAt = now
	if err = s.Supporters.Update(ctx, &supporter); err != nil {
		return SupporterView{}, err
	}
	return s.enrichView(ctx, supporter)
}

func (s *SupporterUseCase) ChangeBilling(ctx context.Context, userID, supporterID uuid.UUID, in ChangeBillingIn) (SupporterView, error) {
	var (
		supporter model.ProjectSupporter
		mode      model.BillingMode
		now       time.Time
		err       error
	)
	mode = model.BillingMode(in.BillingMode)
	if !mode.Valid() {
		return SupporterView{}, ErrBadRequest
	}
	now = time.Now().UTC()
	supporter, err = s.loadOwned(ctx, userID, supporterID)
	if err != nil {
		return SupporterView{}, err
	}
	if !supporterCanManage(supporter, now) || supporter.Status == model.SupporterStatusCancelled {
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
	if mode == model.BillingModeKonbini &&
		supporter.BillingMode == model.BillingModeAuto &&
		supporter.StripeSubscriptionID != nil &&
		s.Stripe != nil {
		if err = s.Stripe.CancelSubscriptionAtPeriodEnd(ctx, *supporter.StripeSubscriptionID); err != nil {
			return SupporterView{}, err
		}
	}
	if err = s.Supporters.Update(ctx, &supporter); err != nil {
		return SupporterView{}, err
	}
	return s.enrichView(ctx, supporter)
}

func (s *SupporterUseCase) applyBillingMode(ctx context.Context, supporter model.ProjectSupporter, mode model.BillingMode) error {
	switch mode {
	case model.BillingModeKonbini:
		supporter.StripeSubscriptionID = nil
		return nil
	case model.BillingModeAuto:
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

func supporterCanManage(s model.ProjectSupporter, now time.Time) bool {
	switch s.Status {
	case model.SupporterStatusActive, model.SupporterStatusPendingRenewal, model.SupporterStatusCancelled:
		return s.HasBenefits(now)
	default:
		return false
	}
}

func (s *SupporterUseCase) ViewForSupporter(ctx context.Context, supporter model.ProjectSupporter) (SupporterView, error) {
	return s.enrichView(ctx, supporter)
}
