package usecase

import (
	"context"

	"github.com/google/uuid"
)

type IConnectUsecase interface {
	GetAccount(ctx context.Context, sellerID uuid.UUID) (*ConnectAccountView, error)
	CreateOnboardingLink(ctx context.Context, sellerID uuid.UUID) (string, error)
}

type IPaymentUsecase interface {
	CreateCheckout(ctx context.Context, in CheckoutIn) (CheckoutOut, error)
	CreateIntent(ctx context.Context, in IntentIn) (IntentOut, error)
	VerifyCheckout(ctx context.Context, buyerID uuid.UUID, sessionID string) (SupporterView, error)
	VerifyIntent(ctx context.Context, buyerID uuid.UUID, intentID string) (SupporterView, error)
	HandleWebhook(ctx context.Context, payload []byte, signature string) error
}

type IPurchaseUsecase interface {
	ListMine(ctx context.Context, buyerID uuid.UUID, limit, offset int) ([]PurchaseView, int64, error)
}

type ISupporterUsecase interface {
	ListMine(ctx context.Context, userID uuid.UUID) ([]SupporterView, error)
	Cancel(ctx context.Context, userID, supporterID uuid.UUID) (SupporterView, error)
	Rejoin(ctx context.Context, userID, supporterID uuid.UUID) (SupporterView, error)
	ChangeBilling(ctx context.Context, userID, supporterID uuid.UUID, in ChangeBillingIn) (SupporterView, error)
}
