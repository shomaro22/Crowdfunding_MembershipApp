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

type SupporterView struct {
	Supporter   *entity.ProjectSupporter
	Project     *entity.Project
	Plan        *entity.ProjectPlan
	PendingPlan *entity.ProjectPlan
}

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

type PurchaseView struct {
	Purchase *entity.Purchase
	Project  *entity.Project
	Buyer    *entity.User
}

type PurchaseUseCase struct {
	Users     repository.UserRepo
	Projects  repository.ProjectRepo
	Purchases repository.PurchaseRepo
}

type ChangeBillingIn struct {
	BillingMode   entity.BillingMode
	PaymentMethod entity.PaymentMethod
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

func (p *PurchaseUseCase) ListMine(ctx context.Context, buyerID uuid.UUID, limit, offset int) ([]PurchaseView, int64, error) {
	var (
		rows  []entity.Purchase
		total int64
		out   []PurchaseView
		err   error
		i     int
	)
	rows, total, err = p.Purchases.ListByBuyerID(ctx, buyerID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out = make([]PurchaseView, 0, len(rows))
	for i = range rows {
		var v PurchaseView
		v, err = p.enrichPurchase(ctx, &rows[i])
		if err != nil {
			return nil, 0, err
		}
		out = append(out, v)
	}
	return out, total, nil
}

func (p *PurchaseUseCase) enrichPurchase(ctx context.Context, row *entity.Purchase) (PurchaseView, error) {
	var (
		v   PurchaseView
		err error
	)
	v.Purchase = row
	v.Project, _, _, err = p.Projects.FindID(ctx, row.ProjectID)
	if err != nil {
		return PurchaseView{}, err
	}
	v.Buyer, err = p.Users.FindID(ctx, row.BuyerID)
	if err != nil {
		return PurchaseView{}, err
	}
	return v, nil
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
