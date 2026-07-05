package stripe

import (
	"context"

	stripesdk "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/paymentintent"
	"github.com/stripe/stripe-go/v82/subscription"
)

type PaymentIntentSnapshot struct {
	ID     string
	Status string
}

func (c *client) CancelSubscriptionAtPeriodEnd(ctx context.Context, subID string) error {
	_, err := subscription.Update(subID, &stripesdk.SubscriptionParams{
		CancelAtPeriodEnd: stripesdk.Bool(true),
	})
	_ = ctx
	return err
}

func (c *client) ResumeSubscription(ctx context.Context, subID string) error {
	_, err := subscription.Update(subID, &stripesdk.SubscriptionParams{
		CancelAtPeriodEnd: stripesdk.Bool(false),
	})
	_ = ctx
	return err
}

func (c *client) GetPaymentIntent(ctx context.Context, intentID string) (PaymentIntentSnapshot, error) {
	var (
		pi  *stripesdk.PaymentIntent
		err error
	)
	_ = ctx
	pi, err = paymentintent.Get(intentID, nil)
	if err != nil {
		return PaymentIntentSnapshot{}, err
	}
	return PaymentIntentSnapshot{ID: pi.ID, Status: string(pi.Status)}, nil
}
