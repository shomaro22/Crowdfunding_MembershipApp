package usecase

import (
	"context"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"
	"crowdfund-app/stripe"

	"github.com/google/uuid"
	stripesdk "github.com/stripe/stripe-go/v82"
)

func isAutoSubscriptionRenewal(inv stripesdk.Invoice) bool {
	switch inv.BillingReason {
	case stripesdk.InvoiceBillingReasonSubscriptionCycle,
		stripesdk.InvoiceBillingReasonSubscriptionThreshold,
		stripesdk.InvoiceBillingReasonSubscriptionUpdate:
		return true
	default:
		return false
	}
}

func invoiceSubscriptionID(inv stripesdk.Invoice) string {
	if inv.Parent != nil &&
		inv.Parent.SubscriptionDetails != nil &&
		inv.Parent.SubscriptionDetails.Subscription != nil {
		return inv.Parent.SubscriptionDetails.Subscription.ID
	}
	return ""
}

func (p *PaymentUseCase) fulfillAutoRenewal(ctx context.Context, inv stripesdk.Invoice) error {
	var (
		subID     string
		supporter *entity.ProjectSupporter
		period    periodRange
		err       error
	)
	subID = invoiceSubscriptionID(inv)
	if subID == "" {
		return nil
	}
	supporter, err = p.Supporters.FindStripeSubscriptionID(ctx, subID)
	if err != nil || supporter == nil {
		return err
	}
	if supporter.BillingMode != entity.BillingModeAuto {
		return nil
	}
	period = periodRange{
		start: time.Unix(inv.PeriodStart, 0).UTC(),
		end:   time.Unix(inv.PeriodEnd, 0).UTC(),
	}
	return repository.WithTx(ctx, p.DB, func(txCtx context.Context) error {
		var (
			now      time.Time
			invID    string
			custID   *string
			purchase entity.Purchase
		)
		now = time.Now().UTC()
		invID = inv.ID
		if inv.Customer != nil {
			custID = stripeString(inv.Customer.ID)
		}
		purchase = entity.Purchase{
			ID:              uuid.New(),
			ProjectID:       supporter.ProjectID,
			SupporterID:     &supporter.ID,
			BuyerID:         supporter.UserID,
			Quantity:        1,
			TotalPrice:      int(inv.AmountPaid),
			Status:          entity.PurchaseStatusPaid,
			PaymentMethod:   entity.PaymentMethodCard,
			PeriodStart:     &period.start,
			PeriodEnd:       &period.end,
			StripeInvoiceID: &invID,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		if err = p.Purchases.Create(txCtx, &purchase); err != nil {
			return err
		}
		if err = p.Purchases.RecordEvent(txCtx, &entity.PurchaseEvent{
			ID:         uuid.New(),
			PurchaseID: purchase.ID,
			ToStatus:   entity.PurchaseStatusPaid,
			CreatedAt:  now,
		}); err != nil {
			return err
		}
		supporter.Status = entity.SupporterStatusActive
		supporter.CurrentPeriodStart = period.start
		supporter.CurrentPeriodEnd = period.end
		supporter.StripeCustomerID = custID
		sub := subID
		supporter.StripeSubscriptionID = &sub
		supporter.UpdatedAt = now
		return p.Supporters.Update(txCtx, supporter)
	})
}

func (p *PaymentUseCase) fulfillInvoiceFirstSupport(ctx context.Context, inv stripesdk.Invoice) error {
	var (
		purchaseIDStr string
		purchase      *entity.Purchase
		err           error
	)
	purchaseIDStr = invoicePurchaseID(inv)
	if purchaseIDStr == "" {
		return nil
	}
	purchase, err = p.Purchases.FindID(ctx, uuid.MustParse(purchaseIDStr))
	if err != nil || purchase == nil {
		return err
	}
	if purchase.Status == entity.PurchaseStatusPaid {
		return p.backfillPaidPurchaseInvoice(ctx, inv, purchase)
	}
	var (
		start = time.Unix(inv.PeriodStart, 0).UTC()
		end   = time.Unix(inv.PeriodEnd, 0).UTC()
		cust  *string
		sub   *string
		invID *string
		subID string
	)
	if inv.Customer != nil {
		cust = stripeString(inv.Customer.ID)
	}
	if subID = invoiceSubscriptionID(inv); subID != "" {
		sub = stripeString(subID)
	}
	invID = stripeString(inv.ID)
	return p.fulfillFirstSupport(ctx, purchase, uuid.Nil, periodRange{start, end}, cust, sub, invID)
}

func invoicePurchaseID(inv stripesdk.Invoice) string {
	var subDetails *stripesdk.InvoiceParentSubscriptionDetails
	if inv.Metadata != nil && inv.Metadata[stripe.MetaPurchaseID] != "" {
		return inv.Metadata[stripe.MetaPurchaseID]
	}
	if inv.Parent != nil {
		subDetails = inv.Parent.SubscriptionDetails
	}
	if subDetails != nil && subDetails.Metadata != nil {
		return subDetails.Metadata[stripe.MetaPurchaseID]
	}
	return ""
}

func (p *PaymentUseCase) backfillPaidPurchaseInvoice(ctx context.Context, inv stripesdk.Invoice, purchase *entity.Purchase) error {
	var (
		supporter *entity.ProjectSupporter
		period    periodRange
		err       error
	)
	if purchase.StripeInvoiceID == nil {
		invID := inv.ID
		purchase.StripeInvoiceID = &invID
		purchase.UpdatedAt = time.Now().UTC()
		if err = p.Purchases.Update(ctx, purchase); err != nil {
			return err
		}
	}
	if purchase.SupporterID == nil {
		return nil
	}
	supporter, err = p.Supporters.FindID(ctx, *purchase.SupporterID)
	if err != nil || supporter == nil {
		return err
	}
	period = periodRange{
		start: time.Unix(inv.PeriodStart, 0).UTC(),
		end:   time.Unix(inv.PeriodEnd, 0).UTC(),
	}
	supporter.CurrentPeriodStart = period.start
	supporter.CurrentPeriodEnd = period.end
	if inv.Customer != nil {
		supporter.StripeCustomerID = stripeString(inv.Customer.ID)
	}
	if subID := invoiceSubscriptionID(inv); subID != "" {
		supporter.StripeSubscriptionID = &subID
	}
	supporter.UpdatedAt = time.Now().UTC()
	return p.Supporters.Update(ctx, supporter)
}

func canIssueKonbiniRenewal(s *entity.ProjectSupporter, now time.Time) bool {
	var windowStart time.Time
	if s == nil {
		return false
	}
	if s.Status == entity.SupporterStatusPendingRenewal {
		return true
	}
	if s.Status != entity.SupporterStatusActive || s.BillingMode != entity.BillingModeKonbini {
		return false
	}
	windowStart = s.CurrentPeriodEnd.AddDate(0, 0, -konbiniRenewalWindowDays)
	return !now.Before(windowStart)
}

const konbiniRenewalWindowDays = 7
