package stripe

import (
	"context"
	"errors"

	stripesdk "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/paymentintent"
	"github.com/stripe/stripe-go/v82/subscription"
)

type UpgradePaymentIn struct {
	Amount           int64
	ConnectAccountID string
	CustomerID       string
	Metadata         map[string]string
}

type UpgradePaymentOut struct {
	PaymentIntentID string
	ClientSecret    string
}

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

func (c *client) UpdateSubscriptionAmount(ctx context.Context, subID string, amount int64) error {
	var (
		sub    *stripesdk.Subscription
		itemID string
		err    error
	)
	_ = ctx
	sub, err = subscription.Get(subID, &stripesdk.SubscriptionParams{
		Expand: stripesdk.StringSlice([]string{"items.data.price.product"}),
	})
	if err != nil {
		return err
	}
	if sub.Items == nil || len(sub.Items.Data) == 0 {
		return errors.New("stripe subscription has no items")
	}
	item := sub.Items.Data[0]
	if item.Price == nil || item.Price.Product == nil {
		return errors.New("stripe subscription missing product")
	}
	itemID = item.ID
	_, err = subscription.Update(subID, &stripesdk.SubscriptionParams{
		Items: []*stripesdk.SubscriptionItemsParams{{
			ID: stripesdk.String(itemID),
			PriceData: &stripesdk.SubscriptionItemPriceDataParams{
				Currency: stripesdk.String(string(stripesdk.CurrencyJPY)),
				Product:  stripesdk.String(item.Price.Product.ID),
				Recurring: &stripesdk.SubscriptionItemPriceDataRecurringParams{
					Interval:      stripesdk.String(string(stripesdk.PriceRecurringIntervalDay)),
					IntervalCount: stripesdk.Int64(30),
				},
				UnitAmount: stripesdk.Int64(amount),
			},
		}},
		ProrationBehavior: stripesdk.String("none"),
	})
	return err
}

func (c *client) CreateUpgradePaymentIntent(ctx context.Context, in UpgradePaymentIn) (UpgradePaymentOut, error) {
	var (
		params *stripesdk.PaymentIntentParams
		pi     *stripesdk.PaymentIntent
		err    error
		out    UpgradePaymentOut
	)
	_ = ctx
	params = &stripesdk.PaymentIntentParams{
		Amount:   stripesdk.Int64(in.Amount),
		Currency: stripesdk.String(string(stripesdk.CurrencyJPY)),
		Metadata: in.Metadata,
		AutomaticPaymentMethods: &stripesdk.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripesdk.Bool(true),
		},
		TransferData: &stripesdk.PaymentIntentTransferDataParams{
			Destination: stripesdk.String(in.ConnectAccountID),
		},
	}
	if in.CustomerID != "" {
		params.Customer = stripesdk.String(in.CustomerID)
	}
	pi, err = paymentintent.New(params)
	if err != nil {
		return UpgradePaymentOut{}, err
	}
	out = UpgradePaymentOut{
		PaymentIntentID: pi.ID,
		ClientSecret:    pi.ClientSecret,
	}
	return out, nil
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
