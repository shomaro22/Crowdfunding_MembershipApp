package usecase

import (
	"context"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"
	"crowdfund-app/stripe"

	"github.com/google/uuid"
)

func (p *PaymentUseCase) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
	var (
		event stripe.WebhookEvent
		err   error
	)
	if p.Stripe == nil {
		return ErrStripeUnavailable
	}
	event, err = p.Stripe.ConstructWebhookEvent(payload, signature)
	if err != nil {
		return err
	}
	var claimed bool
	claimed, err = p.Webhooks.Claim(ctx, event.ID, event.Type)
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}
	switch event.Type {
	case "checkout.session.completed":
		err = p.onCheckoutCompleted(ctx, event)
	case "checkout.session.async_payment_succeeded":
		err = p.onCheckoutAsyncPaid(ctx, event)
	case "checkout.session.async_payment_failed":
		err = p.onCheckoutAsyncFailed(ctx, event)
	case "checkout.session.expired":
		err = p.onCheckoutExpired(ctx, event)
	case "payment_intent.succeeded":
		err = p.onPaymentIntentSucceeded(ctx, event)
	case "invoice.paid":
		err = p.onInvoicePaid(ctx, event)
	case "account.updated":
		err = p.onAccountUpdated(ctx, event)
	default:
		err = nil
	}
	if err != nil {
		_ = p.Webhooks.Release(ctx, event.ID)
		return err
	}
	p.logPaymentWebhook(ctx, event.Type, event.ID)
	return nil
}

func (p *PaymentUseCase) onAccountUpdated(ctx context.Context, event stripe.WebhookEvent) error {
	var (
		acct stripe.AccountEvent
		err  error
	)
	if p.ConnectSync == nil {
		return nil
	}
	acct, err = stripe.ParseWebhookAccount(event)
	if err != nil {
		return err
	}
	_, err = p.ConnectSync.SyncFromStripeAccountID(ctx, acct.AccountID)
	return err
}

func (p *PaymentUseCase) onCheckoutCompleted(ctx context.Context, event stripe.WebhookEvent) error {
	var (
		sess stripe.CheckoutSessionSnapshot
		err  error
	)
	sess, err = stripe.ParseWebhookCheckoutSession(event)
	if err != nil {
		return err
	}
	if isKonbiniSession(sess) {
		return nil
	}
	if sess.PaymentStatus != stripe.CheckoutPaymentStatusPaid {
		return nil
	}
	return p.fulfillCheckoutSession(ctx, sess)
}

func (p *PaymentUseCase) onCheckoutAsyncPaid(ctx context.Context, event stripe.WebhookEvent) error {
	var (
		sess stripe.CheckoutSessionSnapshot
		err  error
	)
	sess, err = stripe.ParseWebhookCheckoutSession(event)
	if err != nil {
		return err
	}
	return p.fulfillCheckoutSession(ctx, sess)
}

func (p *PaymentUseCase) onCheckoutAsyncFailed(ctx context.Context, event stripe.WebhookEvent) error {
	var (
		sess     stripe.CheckoutSessionSnapshot
		purchase *entity.Purchase
		err      error
	)
	sess, err = stripe.ParseWebhookCheckoutSession(event)
	if err != nil {
		return err
	}
	purchase, err = p.Purchases.FindSessionID(ctx, sess.ID)
	if err != nil {
		return err
	}
	return p.handleKonbiniCheckoutUnpaid(ctx, purchase, "async payment failed", false)
}

func (p *PaymentUseCase) onCheckoutExpired(ctx context.Context, event stripe.WebhookEvent) error {
	var (
		sess     stripe.CheckoutSessionSnapshot
		purchase *entity.Purchase
		err      error
	)
	sess, err = stripe.ParseWebhookCheckoutSession(event)
	if err != nil {
		return err
	}
	purchase, err = p.Purchases.FindSessionID(ctx, sess.ID)
	if err != nil {
		return err
	}
	return p.handleKonbiniCheckoutUnpaid(ctx, purchase, "checkout session expired", true)
}

func isKonbiniSession(sess stripe.CheckoutSessionSnapshot) bool {
	var i int
	for i = range sess.PaymentMethodTypes {
		if sess.PaymentMethodTypes[i] == "konbini" {
			return true
		}
	}
	return parseCheckoutPurpose(sess.Metadata) == checkoutKonbiniRenewal ||
		sess.Metadata[stripe.MetaBillingMode] == string(entity.BillingModeKonbini)
}

func (p *PaymentUseCase) onPaymentIntentSucceeded(ctx context.Context, event stripe.WebhookEvent) error {
	var (
		pi  stripe.PaymentIntentDetail
		err error
	)
	pi, err = stripe.ParseWebhookPaymentIntent(event)
	if err != nil {
		return err
	}
	return p.fulfillPaymentIntent(ctx, pi)
}

func (p *PaymentUseCase) onInvoicePaid(ctx context.Context, event stripe.WebhookEvent) error {
	var (
		inv    stripe.InvoiceSnapshot
		exists *entity.Purchase
		err    error
	)
	inv, err = stripe.ParseWebhookInvoice(event)
	if err != nil {
		return err
	}
	exists, err = p.Purchases.FindStripeInvoiceID(ctx, inv.ID)
	if err != nil {
		return err
	}
	if exists != nil {
		return nil
	}
	if stripe.IsAutoSubscriptionRenewal(inv.BillingReason) {
		return p.fulfillAutoRenewal(ctx, inv)
	}
	return p.fulfillInvoiceFirstSupport(ctx, inv)
}

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
