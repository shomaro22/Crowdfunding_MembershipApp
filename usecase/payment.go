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

const supportPeriodDays = 30

type PaymentUseCase struct {
	DB              *gorm.DB
	Users           repository.UserRepo
	Projects        repository.ProjectRepo
	Purchases       repository.PurchaseRepo
	Supporters      repository.SupporterRepo
	Connect         repository.ConnectRepo
	Webhooks        repository.WebhookEventRepo
	Activity        repository.ActivityRepo
	Stripe          stripe.Client
	ConnectSync     ConnectSyncer
	SupporterViewer SupporterViewer
}

type CheckoutIn struct {
	BuyerID            uuid.UUID
	ProjectID          uuid.UUID
	PlanID             uuid.UUID
	PaymentMethod      string
	SuccessURL         *string
	CancelURL          *string
	FallbackSuccessURL string
	FallbackCancelURL  string
	Meta               RequestMeta
}

type CheckoutOut struct {
	SessionID string
	URL       string
}

type IntentIn struct {
	BuyerID   uuid.UUID
	ProjectID uuid.UUID
	PlanID    uuid.UUID
	Meta      RequestMeta
}

type IntentOut struct {
	ClientSecret    string
	PaymentIntentID string
}

type supportContext struct {
	buyer      *entity.User
	project    *entity.Project
	plan       *entity.ProjectPlan
	connect    *entity.ConnectAccount
	membership *entity.ProjectSupporter
	purpose    checkoutPurpose
	now        time.Time
}

type checkoutPurpose string

const (
	checkoutFirstSupport   checkoutPurpose = stripe.PurposeFirstSupport
	checkoutKonbiniRenewal checkoutPurpose = stripe.PurposeKonbiniRenewal
)

type periodRange struct {
	start time.Time
	end   time.Time
}

func (p *PaymentUseCase) logPurchaseCreated(ctx context.Context, purchase *entity.Purchase, meta RequestMeta) {
	var uid *uuid.UUID
	if purchase != nil {
		uid = &purchase.BuyerID
	}
	LogActivity(ctx, p.Activity, uid, entity.PurchaseCreated, "purchase", purchase.ID.String(), meta, purchaseActivityMeta(purchase))
}

func (p *PaymentUseCase) logPaymentCheckoutStarted(ctx context.Context, purchase *entity.Purchase, meta RequestMeta, flow string) {
	var uid *uuid.UUID
	if purchase != nil {
		uid = &purchase.BuyerID
	}
	fields := map[string]string{"flow": flow}
	if purchase != nil {
		fields["project_id"] = purchase.ProjectID.String()
	}
	LogActivity(ctx, p.Activity, uid, entity.PaymentCheckout, "purchase", purchase.ID.String(), meta, activityMeta(fields))
}

func (p *PaymentUseCase) logPurchaseCancelled(ctx context.Context, purchase *entity.Purchase, reason string, to entity.PurchaseStatus) {
	var uid *uuid.UUID
	if purchase != nil {
		uid = &purchase.BuyerID
	}
	LogActivity(ctx, p.Activity, uid, entity.PurchaseCancelled, "purchase", purchase.ID.String(), RequestMeta{}, activityMeta(map[string]string{
		"reason":    reason,
		"to_status": string(to),
	}))
}

func (p *PaymentUseCase) logPaymentWebhook(ctx context.Context, eventType, eventID string) {
	LogActivity(ctx, p.Activity, nil, entity.PaymentWebhook, "stripe_event", eventID, RequestMeta{}, activityMeta(map[string]string{
		"event_type": eventType,
	}))
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

func (p *PaymentUseCase) savePendingAfterStripe(ctx context.Context, purchase *entity.Purchase) error {
	return repository.WithTx(ctx, p.DB, func(txCtx context.Context) error {
		return p.Purchases.Create(txCtx, purchase)
	})
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
