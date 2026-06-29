package stripe

import (
	"encoding/json"

	stripesdk "github.com/stripe/stripe-go/v82"
)

const (
	CheckoutPaymentStatusPaid   = "paid"
	CheckoutPaymentStatusUnpaid = "unpaid"

	PaymentIntentStatusSucceeded = "succeeded"

	InvoiceBillingReasonSubscriptionCycle     = "subscription_cycle"
	InvoiceBillingReasonSubscriptionThreshold = "subscription_threshold"
	InvoiceBillingReasonSubscriptionUpdate    = "subscription_update"
)

type WebhookEvent struct {
	ID      string
	Type    string
	DataRaw json.RawMessage
}

type AccountEvent struct {
	AccountID string
}

type CheckoutSessionSnapshot struct {
	ID                 string
	PaymentStatus      string
	PaymentMethodTypes []string
	Metadata           map[string]string
}

type PaymentIntentDetail struct {
	ID       string
	Status   string
	Metadata map[string]string
}

type InvoiceSnapshot struct {
	ID               string
	BillingReason    string
	PeriodStart      int64
	PeriodEnd        int64
	AmountPaid       int64
	CustomerID       string
	SubscriptionID   string
	Metadata         map[string]string
	SubscriptionMeta map[string]string
}

func ParseWebhookAccount(ev WebhookEvent) (AccountEvent, error) {
	var acct stripesdk.Account
	if err := json.Unmarshal(ev.DataRaw, &acct); err != nil {
		return AccountEvent{}, err
	}
	return AccountEvent{AccountID: acct.ID}, nil
}

func ParseWebhookCheckoutSession(ev WebhookEvent) (CheckoutSessionSnapshot, error) {
	var sess stripesdk.CheckoutSession
	if err := json.Unmarshal(ev.DataRaw, &sess); err != nil {
		return CheckoutSessionSnapshot{}, err
	}
	return checkoutSessionFromSDK(sess), nil
}

func ParseWebhookPaymentIntent(ev WebhookEvent) (PaymentIntentDetail, error) {
	var pi stripesdk.PaymentIntent
	if err := json.Unmarshal(ev.DataRaw, &pi); err != nil {
		return PaymentIntentDetail{}, err
	}
	return paymentIntentFromSDK(pi), nil
}

func ParseWebhookInvoice(ev WebhookEvent) (InvoiceSnapshot, error) {
	var inv stripesdk.Invoice
	if err := json.Unmarshal(ev.DataRaw, &inv); err != nil {
		return InvoiceSnapshot{}, err
	}
	return invoiceFromSDK(inv), nil
}

func checkoutSessionFromSDK(sess stripesdk.CheckoutSession) CheckoutSessionSnapshot {
	return CheckoutSessionSnapshot{
		ID:                 sess.ID,
		PaymentStatus:      string(sess.PaymentStatus),
		PaymentMethodTypes: append([]string(nil), sess.PaymentMethodTypes...),
		Metadata:           cloneStringMap(sess.Metadata),
	}
}

func paymentIntentFromSDK(pi stripesdk.PaymentIntent) PaymentIntentDetail {
	return PaymentIntentDetail{
		ID:       pi.ID,
		Status:   string(pi.Status),
		Metadata: cloneStringMap(pi.Metadata),
	}
}

func invoiceFromSDK(inv stripesdk.Invoice) InvoiceSnapshot {
	var (
		out InvoiceSnapshot
	)
	out = InvoiceSnapshot{
		ID:            inv.ID,
		BillingReason: string(inv.BillingReason),
		PeriodStart:   inv.PeriodStart,
		PeriodEnd:     inv.PeriodEnd,
		AmountPaid:    inv.AmountPaid,
		Metadata:      cloneStringMap(inv.Metadata),
	}
	if inv.Customer != nil {
		out.CustomerID = inv.Customer.ID
	}
	if inv.Parent != nil && inv.Parent.SubscriptionDetails != nil {
		if inv.Parent.SubscriptionDetails.Subscription != nil {
			out.SubscriptionID = inv.Parent.SubscriptionDetails.Subscription.ID
		}
		out.SubscriptionMeta = cloneStringMap(inv.Parent.SubscriptionDetails.Metadata)
	}
	return out
}

func cloneStringMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func (inv InvoiceSnapshot) PurchaseID() string {
	if inv.Metadata != nil && inv.Metadata[MetaPurchaseID] != "" {
		return inv.Metadata[MetaPurchaseID]
	}
	if inv.SubscriptionMeta != nil {
		return inv.SubscriptionMeta[MetaPurchaseID]
	}
	return ""
}

func IsAutoSubscriptionRenewal(billingReason string) bool {
	switch billingReason {
	case InvoiceBillingReasonSubscriptionCycle,
		InvoiceBillingReasonSubscriptionThreshold,
		InvoiceBillingReasonSubscriptionUpdate:
		return true
	default:
		return false
	}
}
