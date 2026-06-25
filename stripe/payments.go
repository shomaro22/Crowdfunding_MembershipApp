package stripe

import (
	"context"
	"errors"

	stripesdk "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/customer"
	"github.com/stripe/stripe-go/v82/product"
	"github.com/stripe/stripe-go/v82/subscription"
)

type CheckoutSessionIn struct {
	ProductName      string
	Amount           int64
	ConnectAccountID string
	CustomerEmail    string
	SuccessURL       string
	CancelURL        string
	PaymentMethod    string // konbini or card
	Metadata         map[string]string
}

type CheckoutSessionOut struct {
	SessionID       string
	URL             string
	PaymentIntentID string
}

type SubscriptionIntentIn struct {
	ProductName      string
	Amount           int64
	ConnectAccountID string
	CustomerEmail    string
	Metadata         map[string]string
}

type SubscriptionIntentOut struct {
	SubscriptionID  string
	PaymentIntentID string
	ClientSecret    string
	CustomerID      string
}

func (c *client) CreateCheckoutSession(ctx context.Context, in CheckoutSessionIn) (CheckoutSessionOut, error) {
	var (
		params *stripesdk.CheckoutSessionParams
		sess   *stripesdk.CheckoutSession
		err    error
		out    CheckoutSessionOut
		pm     []string
	)
	_ = ctx
	if in.PaymentMethod == "konbini" {
		pm = []string{"konbini"}
	} else {
		pm = []string{"card"}
	}
	params = &stripesdk.CheckoutSessionParams{
		Mode:               stripesdk.String(string(stripesdk.CheckoutSessionModePayment)),
		SuccessURL:         stripesdk.String(in.SuccessURL),
		CancelURL:          stripesdk.String(in.CancelURL),
		CustomerEmail:      stripesdk.String(in.CustomerEmail),
		PaymentMethodTypes: stripesdk.StringSlice(pm),
		LineItems: []*stripesdk.CheckoutSessionLineItemParams{{
			PriceData: &stripesdk.CheckoutSessionLineItemPriceDataParams{
				Currency: stripesdk.String(string(stripesdk.CurrencyJPY)),
				ProductData: &stripesdk.CheckoutSessionLineItemPriceDataProductDataParams{
					Name: stripesdk.String(in.ProductName),
				},
				UnitAmount: stripesdk.Int64(in.Amount),
			},
			Quantity: stripesdk.Int64(1),
		}},
		PaymentIntentData: &stripesdk.CheckoutSessionPaymentIntentDataParams{
			Metadata: in.Metadata,
			TransferData: &stripesdk.CheckoutSessionPaymentIntentDataTransferDataParams{
				Destination: stripesdk.String(in.ConnectAccountID),
			},
		},
		Metadata: in.Metadata,
	}
	sess, err = session.New(params)
	if err != nil {
		return CheckoutSessionOut{}, err
	}
	out = CheckoutSessionOut{SessionID: sess.ID, URL: sess.URL}
	if sess.PaymentIntent != nil {
		out.PaymentIntentID = sess.PaymentIntent.ID
	}
	return out, nil
}

func (c *client) CreateSubscriptionIntent(ctx context.Context, in SubscriptionIntentIn) (SubscriptionIntentOut, error) {
	var (
		cust   *stripesdk.Customer
		prod   *stripesdk.Product
		sub    *stripesdk.Subscription
		params *stripesdk.SubscriptionParams
		err    error
		out    SubscriptionIntentOut
		secret string
		piID   string
	)
	_ = ctx
	cust, err = customer.New(&stripesdk.CustomerParams{
		Email: stripesdk.String(in.CustomerEmail),
	})
	if err != nil {
		return SubscriptionIntentOut{}, err
	}
	prod, err = product.New(&stripesdk.ProductParams{
		Name: stripesdk.String(in.ProductName),
	})
	if err != nil {
		return SubscriptionIntentOut{}, err
	}
	params = &stripesdk.SubscriptionParams{
		Customer: stripesdk.String(cust.ID),
		Items: []*stripesdk.SubscriptionItemsParams{{
			PriceData: &stripesdk.SubscriptionItemPriceDataParams{
				Currency: stripesdk.String(string(stripesdk.CurrencyJPY)),
				Product:  stripesdk.String(prod.ID),
				Recurring: &stripesdk.SubscriptionItemPriceDataRecurringParams{
					Interval:      stripesdk.String(string(stripesdk.PriceRecurringIntervalDay)),
					IntervalCount: stripesdk.Int64(30),
				},
				UnitAmount: stripesdk.Int64(in.Amount),
			},
		}},
		PaymentBehavior: stripesdk.String("default_incomplete"),
		PaymentSettings: &stripesdk.SubscriptionPaymentSettingsParams{
			SaveDefaultPaymentMethod: stripesdk.String("on_subscription"),
		},
		TransferData: &stripesdk.SubscriptionTransferDataParams{
			Destination: stripesdk.String(in.ConnectAccountID),
		},
		Metadata: in.Metadata,
		Expand: stripesdk.StringSlice([]string{
			"latest_invoice.confirmation_secret",
			"latest_invoice.payments.data.payment.payment_intent",
		}),
	}
	sub, err = subscription.New(params)
	if err != nil {
		return SubscriptionIntentOut{}, err
	}
	if sub.LatestInvoice != nil && sub.LatestInvoice.ConfirmationSecret != nil {
		secret = sub.LatestInvoice.ConfirmationSecret.ClientSecret
	}
	if sub.LatestInvoice != nil && sub.LatestInvoice.Payments != nil && len(sub.LatestInvoice.Payments.Data) > 0 {
		pay := sub.LatestInvoice.Payments.Data[0].Payment
		if pay != nil && pay.PaymentIntent != nil {
			piID = pay.PaymentIntent.ID
			if secret == "" {
				secret = pay.PaymentIntent.ClientSecret
			}
		}
	}
	if secret == "" {
		return SubscriptionIntentOut{}, errors.New("stripe subscription missing payment secret")
	}
	out = SubscriptionIntentOut{
		SubscriptionID:  sub.ID,
		PaymentIntentID: piID,
		ClientSecret:    secret,
		CustomerID:      cust.ID,
	}
	return out, nil
}
