package usecase

import (
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"
	"crowdfund-app/stripe"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const supportPeriodDays = 30

type PaymentUseCase struct {
	DB         *gorm.DB
	Users      repository.UserRepo
	Projects   repository.ProjectRepo
	Purchases  repository.PurchaseRepo
	Supporters repository.SupporterRepo
	Connect    repository.ConnectRepo
	Webhooks   repository.WebhookEventRepo
	Stripe     stripe.Client
	ConnectUC  *ConnectUseCase
}

type CheckoutIn struct {
	BuyerID       uuid.UUID
	ProjectID     uuid.UUID
	PlanID        uuid.UUID
	PaymentMethod entity.PaymentMethod
	SuccessURL    *string
	CancelURL     *string
}

type CheckoutOut struct {
	SessionID string
	URL       string
}

type IntentIn struct {
	BuyerID   uuid.UUID
	ProjectID uuid.UUID
	PlanID    uuid.UUID
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
