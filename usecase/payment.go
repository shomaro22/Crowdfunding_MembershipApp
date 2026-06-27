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
	BuyerID       uuid.UUID
	ProjectID     uuid.UUID
	PlanID        uuid.UUID
	PaymentMethod string
	SuccessURL    *string
	CancelURL     *string
	Meta          RequestMeta
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
