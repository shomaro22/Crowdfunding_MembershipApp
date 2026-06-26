package stripe

import (
	"context"

	stripesdk "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/paymentintent"
)

func (c *client) GetCheckoutSession(ctx context.Context, sessionID string) (*stripesdk.CheckoutSession, error) {
	_ = ctx
	return session.Get(sessionID, nil)
}

func (c *client) GetPaymentIntentDetail(ctx context.Context, intentID string) (*stripesdk.PaymentIntent, error) {
	_ = ctx
	return paymentintent.Get(intentID, nil)
}
