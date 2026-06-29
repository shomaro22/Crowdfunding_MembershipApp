package stripe

import (
	"context"

	stripesdk "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/paymentintent"
)

func (c *client) GetCheckoutSession(ctx context.Context, sessionID string) (CheckoutSessionSnapshot, error) {
	var (
		sess *stripesdk.CheckoutSession
		err  error
	)
	_ = ctx
	sess, err = session.Get(sessionID, nil)
	if err != nil {
		return CheckoutSessionSnapshot{}, err
	}
	return checkoutSessionFromSDK(*sess), nil
}

func (c *client) GetPaymentIntentDetail(ctx context.Context, intentID string) (PaymentIntentDetail, error) {
	var (
		pi  *stripesdk.PaymentIntent
		err error
	)
	_ = ctx
	pi, err = paymentintent.Get(intentID, nil)
	if err != nil {
		return PaymentIntentDetail{}, err
	}
	return paymentIntentFromSDK(*pi), nil
}
