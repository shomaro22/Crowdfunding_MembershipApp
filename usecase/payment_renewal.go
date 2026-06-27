package usecase

import (
	"context"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"
	"crowdfund-app/stripe"

	"github.com/google/uuid"
)

func (p *PaymentUseCase) fulfillAutoRenewal(ctx context.Context, inv stripe.InvoiceSnapshot) error {
	var (
		subID     string
		supporter *entity.ProjectSupporter
		period    periodRange
		err       error
	)
	subID = inv.SubscriptionID
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
		if inv.CustomerID != "" {
			custID = stripeString(inv.CustomerID)
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
		LogActivity(txCtx, p.Activity, &supporter.UserID, entity.PurchaseCreated, "purchase", purchase.ID.String(), RequestMeta{}, activityMeta(map[string]string{
			"project_id": supporter.ProjectID.String(),
			"renewal":    "true",
		}))
		if err = p.Projects.AddCurrentAmount(txCtx, supporter.ProjectID, inv.AmountPaid); err != nil {
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

func (p *PaymentUseCase) fulfillInvoiceFirstSupport(ctx context.Context, inv stripe.InvoiceSnapshot) error {
	var (
		purchaseIDStr string
		purchase      *entity.Purchase
		err           error
	)
	purchaseIDStr = inv.PurchaseID()
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
	if inv.CustomerID != "" {
		cust = stripeString(inv.CustomerID)
	}
	if subID = inv.SubscriptionID; subID != "" {
		sub = stripeString(subID)
	}
	invID = stripeString(inv.ID)
	return p.fulfillFirstSupport(ctx, purchase, uuid.Nil, periodRange{start, end}, cust, sub, invID)
}

func (p *PaymentUseCase) backfillPaidPurchaseInvoice(ctx context.Context, inv stripe.InvoiceSnapshot, purchase *entity.Purchase) error {
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
	if inv.CustomerID != "" {
		supporter.StripeCustomerID = stripeString(inv.CustomerID)
	}
	if inv.SubscriptionID != "" {
		supporter.StripeSubscriptionID = &inv.SubscriptionID
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
