package usecase

import (
	"context"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/stripe"

	"github.com/google/uuid"
)

func (p *PaymentUseCase) loadSupportContext(ctx context.Context, buyerID, projectID, planID uuid.UUID) (supportContext, error) {
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
	if err != nil {
		return sc, err
	}
	if pend != nil {
		return sc, ErrPendingCheckout
	}
	if sc.membership, err = p.Supporters.FindMembership(ctx, projectID, buyerID); err != nil {
		return sc, err
	}
	if sc.membership != nil {
		return sc, ErrAlreadySupporting
	}
	if sc.plan.MaxSupporters != nil && sc.plan.SupporterCount >= *sc.plan.MaxSupporters {
		return sc, ErrForbidden
	}
	sc.purpose = checkoutFirstSupport
	return sc, nil
}

func (p *PaymentUseCase) loadCheckoutContext(ctx context.Context, buyerID, projectID, planID uuid.UUID, pm entity.PaymentMethod) (supportContext, error) {
	var (
		sc   supportContext
		pend *entity.Purchase
		err  error
	)
	if pm == entity.PaymentMethodKonbini {
		return p.loadKonbiniCheckoutContext(ctx, buyerID, projectID, planID)
	}
	sc, err = p.loadBuyerProjectPlan(ctx, buyerID, projectID, planID)
	if err != nil {
		return sc, err
	}
	pend, err = p.Purchases.FindPendingBuyerProject(ctx, buyerID, projectID)
	if err != nil {
		return sc, err
	}
	if pend != nil {
		return sc, ErrPendingCheckout
	}
	if sc.membership, err = p.Supporters.FindMembership(ctx, projectID, buyerID); err != nil {
		return sc, err
	}
	if sc.membership != nil {
		return sc, ErrAlreadySupporting
	}
	if sc.plan.MaxSupporters != nil && sc.plan.SupporterCount >= *sc.plan.MaxSupporters {
		return sc, ErrForbidden
	}
	sc.purpose = checkoutFirstSupport
	return sc, nil
}

func (p *PaymentUseCase) loadKonbiniCheckoutContext(ctx context.Context, buyerID, projectID, planID uuid.UUID) (supportContext, error) {
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
	if err != nil {
		return sc, err
	}
	if pend != nil {
		return sc, ErrPendingCheckout
	}
	sc.membership, err = p.Supporters.FindMembership(ctx, projectID, buyerID)
	if err != nil {
		return sc, err
	}
	if sc.membership == nil {
		if sc.plan.MaxSupporters != nil && sc.plan.SupporterCount >= *sc.plan.MaxSupporters {
			return sc, ErrForbidden
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

func (p *PaymentUseCase) loadBuyerProjectPlan(ctx context.Context, buyerID, projectID, planID uuid.UUID) (supportContext, error) {
	var (
		sc   supportContext
		u    *entity.User
		project *entity.Project
		plan *entity.ProjectPlan
		conn *entity.ConnectAccount
		err  error
	)
	sc.now = time.Now().UTC()
	u, err = p.Users.FindID(ctx, buyerID)
	if err != nil || u == nil {
		return sc, ErrNotFound
	}
	if !u.IsActive() || u.Role != entity.UserRoleBuyer {
		return sc, ErrForbidden
	}
	project, _, _, err = p.Projects.FindID(ctx, projectID)
	if err != nil || project == nil {
		return sc, ErrNotFound
	}
	if !project.IsPublic(sc.now) {
		return sc, ErrForbidden
	}
	plan, err = p.Projects.FindPlan(ctx, projectID, planID)
	if err != nil || plan == nil || !plan.IsActive {
		return sc, ErrNotFound
	}
	conn, err = p.Connect.FindUserID(ctx, project.SellerID)
	if err != nil {
		return sc, err
	}
	if conn == nil || !conn.CanCharge() {
		return sc, ErrForbidden
	}
	sc.buyer = u
	sc.project = project
	sc.plan = plan
	sc.connect = conn
	return sc, nil
}

func (p *PaymentUseCase) newPendingPurchase(
	sc supportContext,
	pm entity.PaymentMethod,
	checkoutID, intentID *string,
) *entity.Purchase {
	var (
		row entity.Purchase
		key string
	)
	key = uuid.New().String()
	row = entity.Purchase{
		ID:               uuid.New(),
		ProjectID:        sc.project.ID,
		BuyerID:          sc.buyer.ID,
		Quantity:         1,
		TotalPrice:       sc.plan.Price,
		Status:           entity.PurchaseStatusPending,
		PaymentMethod:    pm,
		StripeCheckoutID: checkoutID,
		StripeIntentID:   intentID,
		IdempotencyKey:   &key,
		CreatedAt:        sc.now,
		UpdatedAt:        sc.now,
	}
	if sc.purpose == checkoutKonbiniRenewal && sc.membership != nil {
		row.SupporterID = &sc.membership.ID
	}
	return &row
}

func (p *PaymentUseCase) createPendingPurchase(
	ctx context.Context,
	sc supportContext,
	pm entity.PaymentMethod,
	checkoutID, intentID *string,
) (*entity.Purchase, error) {
	var (
		row *entity.Purchase
		err error
	)
	row = p.newPendingPurchase(sc, pm, checkoutID, intentID)
	err = p.Purchases.Create(ctx, row)
	if err != nil {
		return nil, err
	}
	return row, nil
}

func checkoutMetadata(sc supportContext, purchaseID uuid.UUID, pm entity.PaymentMethod) map[string]string {
	var mode entity.BillingMode
	if pm == entity.PaymentMethodKonbini {
		mode = entity.BillingModeKonbini
	} else {
		mode = entity.BillingModeAuto
	}
	meta := paymentMetadata(
		purchaseID,
		sc.project.ID,
		sc.plan.ID,
		sc.buyer.ID,
		mode,
	)
	meta[stripe.MetaCheckoutPurpose] = string(sc.purpose)
	if sc.membership != nil {
		meta[stripe.MetaSupporterID] = sc.membership.ID.String()
	}
	return meta
}

func paymentMetadata(purchaseID, projectID, planID, buyerID uuid.UUID, mode entity.BillingMode) map[string]string {
	return map[string]string{
		stripe.MetaPurchaseID:  purchaseID.String(),
		stripe.MetaProjectID:   projectID.String(),
		stripe.MetaPlanID:      planID.String(),
		stripe.MetaBuyerID:     buyerID.String(),
		stripe.MetaBillingMode: string(mode),
	}
}

func checkoutURL(custom *string, fallback string) string {
	if custom != nil && *custom != "" {
		return *custom
	}
	return fallback
}

func parseCheckoutPurpose(meta map[string]string) checkoutPurpose {
	if meta == nil {
		return checkoutFirstSupport
	}
	switch meta[stripe.MetaCheckoutPurpose] {
	case stripe.PurposeKonbiniRenewal:
		return checkoutKonbiniRenewal
	default:
		return checkoutFirstSupport
	}
}

func isKonbiniPurchase(purchase *entity.Purchase) bool {
	return purchase.PaymentMethod == entity.PaymentMethodKonbini
}
