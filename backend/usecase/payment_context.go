package usecase

import (
	"context"
	"time"

	"crowdfund-app/model"

	"github.com/google/uuid"
)

func (p *PaymentUseCase) supporterViewForPurchase(ctx context.Context, purchase model.Purchase) (SupporterView, error) {
	var (
		supporter model.ProjectSupporter
		err       error
	)
	if purchase.SupporterID != nil {
		supporter, err = p.Supporters.GetID(ctx, *purchase.SupporterID)
	} else {
		supporter, err = p.Supporters.GetActive(ctx, purchase.ProjectID, purchase.BuyerID)
	}
	if err != nil {
		return SupporterView{}, mapRepoNotFound(err)
	}
	if p.SupporterViewer == nil {
		return SupporterView{}, ErrNotFound
	}
	return p.SupporterViewer.ViewForSupporter(ctx, supporter)
}

func (p *PaymentUseCase) loadSupportContext(ctx context.Context, buyerID, projectID, planID uuid.UUID) (supportContext, error) {
	var (
		sc  supportContext
		err error
	)
	sc, err = p.loadCheckoutBase(ctx, buyerID, projectID, planID)
	if err != nil {
		return sc, err
	}
	sc.membership, err = p.Supporters.GetMembership(ctx, projectID, buyerID)
	if isRepoNotFound(err) {
		sc.hasMembership = false
	} else if err != nil {
		return sc, err
	} else {
		sc.hasMembership = true
	}
	if sc.hasMembership {
		return sc, ErrAlreadySupporting
	}
	if err = checkPlanCapacity(sc.plan); err != nil {
		return sc, err
	}
	sc.purpose = checkoutFirstSupport
	return sc, nil
}

func (p *PaymentUseCase) loadCheckoutContext(ctx context.Context, buyerID, projectID, planID uuid.UUID, pm model.PaymentMethod) (supportContext, error) {
	if pm == model.PaymentMethodKonbini {
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
	sc.membership, err = p.Supporters.GetMembership(ctx, projectID, buyerID)
	if isRepoNotFound(err) {
		sc.hasMembership = false
	} else if err != nil {
		return sc, err
	} else {
		sc.hasMembership = true
	}
	if !sc.hasMembership {
		if err = checkPlanCapacity(sc.plan); err != nil {
			return sc, err
		}
		sc.purpose = checkoutFirstSupport
		return sc, nil
	}
	if sc.membership.BillingMode != model.BillingModeKonbini {
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
		pend model.Purchase
		err  error
	)
	sc, err = p.loadBuyerProjectPlan(ctx, buyerID, projectID, planID)
	if err != nil {
		return sc, err
	}
	pend, err = p.Purchases.GetPendingBuyerProject(ctx, buyerID, projectID)
	if isRepoNotFound(err) {
		// no pending checkout
	} else if err != nil {
		return sc, err
	} else if pend.ID != uuid.Nil {
		pend.Status = model.PurchaseStatusCancelled
		pend.UpdatedAt = sc.now
		if err = p.Purchases.Update(ctx, &pend); err != nil {
			return sc, err
		}
	}
	return sc, nil
}

func (p *PaymentUseCase) loadBuyerProjectPlan(ctx context.Context, buyerID, projectID, planID uuid.UUID) (supportContext, error) {
	var (
		sc        supportContext
		u         model.User
		project   model.Project
		plans     []model.ProjectPlan
		plan      model.ProjectPlan
		planFound bool
		conn      model.ConnectAccount
		i         int
		err       error
	)
	sc.now = time.Now().UTC()
	u, err = p.Users.GetID(ctx, buyerID)
	if err != nil {
		return sc, mapRepoNotFound(err)
	}
	if !u.IsActive() || u.Role != model.UserRoleBuyer {
		return sc, ErrForbidden
	}
	project, plans, _, err = p.Projects.GetID(ctx, projectID)
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
			plan = plans[i]
			planFound = true
			break
		}
	}
	if !planFound {
		return sc, ErrNotFound
	}
	conn, err = p.Connect.GetUserID(ctx, project.SellerID)
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
