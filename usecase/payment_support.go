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
		sc     supportContext
		u      *entity.User
		proj   *entity.Project
		plan   *entity.ProjectPlan
		active *entity.ProjectSupporter
		conn   *entity.ConnectAccount
		err    error
	)
	sc.now = time.Now().UTC()
	u, err = p.Users.FindByID(ctx, buyerID)
	if err != nil || u == nil {
		return sc, ErrNotFound
	}
	if !u.IsActive() || u.Role != entity.UserRoleBuyer {
		return sc, ErrForbidden
	}
	proj, _, _, err = p.Projects.FindByID(ctx, projectID)
	if err != nil || proj == nil {
		return sc, ErrNotFound
	}
	if !proj.IsPublic(sc.now) {
		return sc, ErrForbidden
	}
	plan, err = p.Projects.FindPlan(ctx, projectID, planID)
	if err != nil || plan == nil || !plan.IsActive {
		return sc, ErrNotFound
	}
	if plan.MaxSupporters != nil && plan.SupporterCount >= *plan.MaxSupporters {
		return sc, ErrForbidden
	}
	active, err = p.Supporters.FindActive(ctx, projectID, buyerID)
	if err != nil {
		return sc, err
	}
	if active != nil {
		return sc, ErrAlreadySupporting
	}
	conn, err = p.Connect.FindByUserID(ctx, proj.SellerID)
	if err != nil {
		return sc, err
	}
	if conn == nil || !conn.CanCharge() {
		return sc, ErrForbidden
	}
	sc.buyer = u
	sc.project = proj
	sc.plan = plan
	sc.connect = conn
	return sc, nil
}

func (p *PaymentUseCase) createPendingPurchase(
	ctx context.Context,
	sc supportContext,
	pm entity.PaymentMethod,
	mode entity.BillingMode,
	checkoutID, intentID *string,
) (*entity.Purchase, error) {
	var (
		row entity.Purchase
		key string
		err error
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
	_ = mode
	err = p.Purchases.Create(ctx, &row)
	if err != nil {
		return nil, err
	}
	return &row, nil
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
