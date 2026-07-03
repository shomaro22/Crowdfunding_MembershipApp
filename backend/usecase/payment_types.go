package usecase

import (
	"time"

	"crowdfund-app/events"
	"crowdfund-app/model"
	"crowdfund-app/repository"
	"crowdfund-app/stripe"

	"github.com/google/uuid"
)

const (
	supportPeriodDays        = 30
	konbiniRenewalWindowDays = 7
)

type PaymentUseCase struct {
	Tx              repository.TxBeginner
	Users           repository.IUserRepo
	Projects        repository.IProjectRepo
	Purchases       repository.IPurchaseRepo
	Supporters      repository.ISupporterRepo
	Connect         repository.IConnectRepo
	Webhooks        repository.IWebhookEventRepo
	Activity        repository.IActivityRepo
	Stripe          stripe.Client
	ConnectSync     ConnectSyncer
	SupporterViewer SupporterViewer
	PlanUpgrades    PlanUpgradeConfirmer
	supportEvents   *events.Bus
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
	buyer         model.User
	project       model.Project
	plan          model.ProjectPlan
	connect       model.ConnectAccount
	membership    model.ProjectSupporter
	hasMembership bool
	purpose       checkoutPurpose
	now           time.Time
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
