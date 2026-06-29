package usecase

import (
	"context"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
)

const konbiniRenewalWindowDays = 7

func (p *PaymentUseCase) loadSupportContext(ctx context.Context, buyerID, projectID, planID uuid.UUID) (supportContext, error) {
	var (
		sc  supportContext
		err error
	)
	sc, err = p.loadCheckoutBase(ctx, buyerID, projectID, planID)
	if err != nil {
		return sc, err
	}
	sc.membership, err = p.Supporters.FindMembership(ctx, projectID, buyerID)
	if isRepoNotFound(err) {
		sc.membership = nil
	} else if err != nil {
		return sc, err
	}
	if sc.membership != nil {
		return sc, ErrAlreadySupporting
	}
	if err = checkPlanCapacity(sc.plan); err != nil {
		return sc, err
	}
	sc.purpose = checkoutFirstSupport
	return sc, nil
}

func (p *PaymentUseCase) loadCheckoutContext(ctx context.Context, buyerID, projectID, planID uuid.UUID, pm entity.PaymentMethod) (supportContext, error) {
	if pm == entity.PaymentMethodKonbini {
		return p.loadKonbiniCheckoutContext(ctx, buyerID, projectID, planID)
	}
	return p.loadSupportContext(ctx, buyerID, projectID, planID)
}

func (p *PaymentUseCase) loadKonbiniCheckoutContext(ctx context.Context, buyerID, projectID, planID uuid.UUID) (supportContext, error) {
	var (
		sc  supportContext
		err error
	)
	sc, err = p.loadCheckoutBase(ctx, buyerID, projectID, planID)
	if err != nil {
		return sc, err
	}
	sc.membership, err = p.Supporters.FindMembership(ctx, projectID, buyerID)
	if isRepoNotFound(err) {
		sc.membership = nil
	} else if err != nil {
		return sc, err
	}
	if sc.membership == nil {
		if err = checkPlanCapacity(sc.plan); err != nil {
			return sc, err
		}
		sc.purpose = checkoutFirstSupport
		return sc, nil
	}
	if sc.membership.BillingMode != entity.BillingModeKonbini {
		return sc, ErrBadRequest
	}
	if !canIssueKonbiniRenewal(sc.membership, sc.now) {
		return sc, ErrBadRequest
	}
	sc.purpose = checkoutKonbiniRenewal
	return sc, nil
}

func (p *PaymentUseCase) loadCheckoutBase(ctx context.Context, buyerID, projectID, planID uuid.UUID) (supportContext, error) {
	var (
		sc   supportContext
		pend *entity.Purchase
		err  error
	)
	sc, err = p.loadBuyerProjectPlan(ctx, buyerID, projectID, planID)
	if err != nil {
		return sc, err
	}
	pend, err = p.Purchases.FindPendingBuyerProject(ctx, buyerID, projectID)
	if isRepoNotFound(err) {
		pend = nil
	} else if err != nil {
		return sc, err
	}
	if pend != nil {
		return sc, ErrPendingCheckout
	}
	return sc, nil
}

func checkPlanCapacity(plan *entity.ProjectPlan) error {
	if plan.MaxSupporters != nil && plan.SupporterCount >= *plan.MaxSupporters {
		return ErrForbidden
	}
	return nil
}

func (p *PaymentUseCase) loadBuyerProjectPlan(ctx context.Context, buyerID, projectID, planID uuid.UUID) (supportContext, error) {
	var (
		sc      supportContext
		u       *entity.User
		project *entity.Project
		plans   []entity.ProjectPlan
		plan    *entity.ProjectPlan
		conn    *entity.ConnectAccount
		i       int
		err     error
	)
	sc.now = time.Now().UTC()
	u, err = p.Users.FindID(ctx, buyerID)
	if err != nil {
		return sc, mapRepoNotFound(err)
	}
	if !u.IsActive() || u.Role != entity.UserRoleBuyer {
		return sc, ErrForbidden
	}
	project, plans, _, err = p.Projects.FindID(ctx, projectID)
	if err != nil {
		return sc, mapRepoNotFound(err)
	}
	if !project.IsPublic(sc.now) {
		return sc, ErrForbidden
	}
	for i = range plans {
		if plans[i].ID == planID {
			if !plans[i].IsActive {
				return sc, ErrNotFound
			}
			plan = &plans[i]
			break
		}
	}
	if plan == nil {
		return sc, ErrNotFound
	}
	conn, err = p.Connect.FindUserID(ctx, project.SellerID)
	if isRepoNotFound(err) {
		return sc, ErrForbidden
	}
	if err != nil {
		return sc, err
	}
	if !conn.CanCharge() {
		return sc, ErrForbidden
	}
	sc.buyer = u
	sc.project = project
	sc.plan = plan
	sc.connect = conn
	return sc, nil
}

func canIssueKonbiniRenewal(s *entity.ProjectSupporter, now time.Time) bool {
	var windowStart time.Time
	if s == nil {
		return false
	}
	if s.Status == entity.SupporterStatusPendingRenewal {
		return true
	}
	if s.Status != entity.SupporterStatusActive || s.BillingMode != entity.BillingModeKonbini {
		return false
	}
	windowStart = s.CurrentPeriodEnd.AddDate(0, 0, -konbiniRenewalWindowDays)
	return !now.Before(windowStart)
}
