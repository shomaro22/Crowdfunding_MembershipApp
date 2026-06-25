package usecase

import (
	"context"
	"encoding/json"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"
	"crowdfund-app/stripe"

	"github.com/google/uuid"
	stripesdk "github.com/stripe/stripe-go/v82"
	"gorm.io/gorm"
)

const supportPeriodDays = 30

type PaymentUseCase struct {
	DB         *gorm.DB
	Users      repository.UserRepo
	Projects   repository.ProjectRepo
	Purchases  repository.PurchaseRepo
	Supporters repository.SupporterRepo
	Connect    repository.ConnectRepo
	Webhooks   repository.WebhookEventRepo
	Stripe     stripe.Client
	ConnectUC  *ConnectUseCase
}

type CheckoutIn struct {
	BuyerID       uuid.UUID
	ProjectID     uuid.UUID
	PlanID        uuid.UUID
	PaymentMethod entity.PaymentMethod
	SuccessURL    *string
	CancelURL     *string
}

type CheckoutOut struct {
	SessionID string
	URL       string
}

type IntentIn struct {
	BuyerID   uuid.UUID
	ProjectID uuid.UUID
	PlanID    uuid.UUID
}

type IntentOut struct {
	ClientSecret    string
	PaymentIntentID string
}

type supportContext struct {
	buyer    *entity.User
	project  *entity.Project
	plan     *entity.ProjectPlan
	connect  *entity.ConnectAccount
	now      time.Time
}

func (p *PaymentUseCase) CreateCheckout(ctx context.Context, in CheckoutIn) (CheckoutOut, error) {
	var (
		sc      supportContext
		purchase *entity.Purchase
		meta    map[string]string
		sess    stripe.CheckoutSessionOut
		pm      entity.PaymentMethod
		err     error
	)
	if p.Stripe == nil {
		return CheckoutOut{}, ErrStripeUnavailable
	}
	pm = in.PaymentMethod
	if pm == "" {
		pm = entity.PaymentMethodKonbini
	}
	if pm != entity.PaymentMethodKonbini && pm != entity.PaymentMethodCard {
		return CheckoutOut{}, ErrBadRequest
	}
	sc, err = p.loadSupportContext(ctx, in.BuyerID, in.ProjectID, in.PlanID)
	if err != nil {
		return CheckoutOut{}, err
	}
	purchase, err = p.createPendingPurchase(ctx, sc, pm, entity.BillingModeKonbini, nil, nil)
	if err != nil {
		return CheckoutOut{}, err
	}
	meta = paymentMetadata(purchase.ID, in.ProjectID, in.PlanID, in.BuyerID, entity.BillingModeKonbini)
	sess, err = p.Stripe.CreateCheckoutSession(ctx, stripe.CheckoutSessionIn{
		ProductName:      sc.project.Title,
		Amount:           int64(sc.plan.Price),
		ConnectAccountID: sc.connect.StripeAccountID,
		CustomerEmail:    sc.buyer.Email,
		SuccessURL:       checkoutURL(in.SuccessURL, p.Stripe.AppURL()+"/support/success?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:        checkoutURL(in.CancelURL, p.Stripe.AppURL()+"/support/cancel"),
		PaymentMethod:    string(pm),
		Metadata:         meta,
	})
	if err != nil {
		return CheckoutOut{}, err
	}
	purchase.StripeCheckoutID = &sess.SessionID
	purchase.UpdatedAt = sc.now
	if err = p.Purchases.Update(ctx, purchase); err != nil {
		return CheckoutOut{}, err
	}
	return CheckoutOut{SessionID: sess.SessionID, URL: sess.URL}, nil
}

func (p *PaymentUseCase) CreateIntent(ctx context.Context, in IntentIn) (IntentOut, error) {
	var (
		sc      supportContext
		purchase *entity.Purchase
		meta    map[string]string
		sub     stripe.SubscriptionIntentOut
		err     error
	)
	if p.Stripe == nil {
		return IntentOut{}, ErrStripeUnavailable
	}
	sc, err = p.loadSupportContext(ctx, in.BuyerID, in.ProjectID, in.PlanID)
	if err != nil {
		return IntentOut{}, err
	}
	purchase, err = p.createPendingPurchase(ctx, sc, entity.PaymentMethodCard, entity.BillingModeAuto, nil, nil)
	if err != nil {
		return IntentOut{}, err
	}
	meta = paymentMetadata(purchase.ID, in.ProjectID, in.PlanID, in.BuyerID, entity.BillingModeAuto)
	sub, err = p.Stripe.CreateSubscriptionIntent(ctx, stripe.SubscriptionIntentIn{
		ProductName:      sc.project.Title,
		Amount:           int64(sc.plan.Price),
		ConnectAccountID: sc.connect.StripeAccountID,
		CustomerEmail:    sc.buyer.Email,
		Metadata:         meta,
	})
	if err != nil {
		return IntentOut{}, err
	}
	purchase.StripeIntentID = &sub.PaymentIntentID
	purchase.UpdatedAt = sc.now
	if err = p.Purchases.Update(ctx, purchase); err != nil {
		return IntentOut{}, err
	}
	return IntentOut{ClientSecret: sub.ClientSecret, PaymentIntentID: sub.PaymentIntentID}, nil
}

func (p *PaymentUseCase) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
	var (
		event stripesdk.Event
		err   error
	)
	if p.Stripe == nil {
		return ErrStripeUnavailable
	}
	event, err = p.Stripe.ConstructWebhookEvent(payload, signature)
	if err != nil {
		return err
	}
	var exists bool
	exists, err = p.Webhooks.Exists(ctx, event.ID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	switch string(event.Type) {
	case "checkout.session.completed":
		err = p.onCheckoutCompleted(ctx, event)
	case "checkout.session.async_payment_succeeded":
		err = p.onCheckoutAsyncPaid(ctx, event)
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
		return err
	}
	return p.Webhooks.Create(ctx, &entity.WebhookEvent{
		ID:            uuid.New(),
		StripeEventID: event.ID,
		EventType:     string(event.Type),
		ProcessedAt:   time.Now().UTC(),
	})
}

func (p *PaymentUseCase) loadSupportContext(ctx context.Context, buyerID, projectID, planID uuid.UUID) (supportContext, error) {
	var (
		sc   supportContext
		u    *entity.User
		proj *entity.Project
		plan *entity.ProjectPlan
		active *entity.ProjectSupporter
		conn *entity.ConnectAccount
		err  error
	)
	sc.now = time.Now().UTC()
	u, err = p.Users.FindByID(ctx, buyerID)
	if err != nil || u == nil {
		return sc, ErrNotFound
	}
	if !u.IsActive() || u.Role != entity.UserRoleBuyer {
		return sc, ErrForbidden
	}
	proj, _, _, err = p.Projects.FindByID(ctx, projectID)
	if err != nil || proj == nil {
		return sc, ErrNotFound
	}
	if !proj.IsPublic(sc.now) {
		return sc, ErrForbidden
	}
	plan, err = p.Projects.FindPlan(ctx, projectID, planID)
	if err != nil || plan == nil || !plan.IsActive {
		return sc, ErrNotFound
	}
	if plan.MaxSupporters != nil && plan.SupporterCount >= *plan.MaxSupporters {
		return sc, ErrForbidden
	}
	active, err = p.Supporters.FindActive(ctx, projectID, buyerID)
	if err != nil {
		return sc, err
	}
	if active != nil {
		return sc, ErrAlreadySupporting
	}
	conn, err = p.Connect.FindByUserID(ctx, proj.SellerID)
	if err != nil {
		return sc, err
	}
	if conn == nil || !conn.CanCharge() {
		return sc, ErrForbidden
	}
	sc.buyer = u
	sc.project = proj
	sc.plan = plan
	sc.connect = conn
	return sc, nil
}

func (p *PaymentUseCase) createPendingPurchase(
	ctx context.Context,
	sc supportContext,
	pm entity.PaymentMethod,
	mode entity.BillingMode,
	checkoutID, intentID *string,
) (*entity.Purchase, error) {
	var (
		row entity.Purchase
		key string
		err error
	)
	key = uuid.New().String()
	row = entity.Purchase{
		ID:            uuid.New(),
		ProjectID:     sc.project.ID,
		BuyerID:       sc.buyer.ID,
		Quantity:      1,
		TotalPrice:    sc.plan.Price,
		Status:        entity.PurchaseStatusPending,
		PaymentMethod: pm,
		StripeCheckoutID: checkoutID,
		StripeIntentID:   intentID,
		IdempotencyKey:   &key,
		CreatedAt:        sc.now,
		UpdatedAt:        sc.now,
	}
	_ = mode
	err = p.Purchases.Create(ctx, &row)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func paymentMetadata(purchaseID, projectID, planID, buyerID uuid.UUID, mode entity.BillingMode) map[string]string {
	return map[string]string{
		stripe.MetaPurchaseID:  purchaseID.String(),
		stripe.MetaProjectID:   projectID.String(),
		stripe.MetaPlanID:      planID.String(),
		stripe.MetaBuyerID:     buyerID.String(),
		stripe.MetaBillingMode: string(mode),
	}
}

func checkoutURL(custom *string, fallback string) string {
	if custom != nil && *custom != "" {
		return *custom
	}
	return fallback
}

func (p *PaymentUseCase) onCheckoutCompleted(ctx context.Context, event stripesdk.Event) error {
	var (
		sess stripesdk.CheckoutSession
		err  error
	)
	err = json.Unmarshal(event.Data.Raw, &sess)
	if err != nil {
		return err
	}
	if sess.PaymentStatus != stripesdk.CheckoutSessionPaymentStatusPaid {
		return nil
	}
	return p.fulfillByCheckoutSession(ctx, sess)
}

func (p *PaymentUseCase) onCheckoutAsyncPaid(ctx context.Context, event stripesdk.Event) error {
	var (
		sess stripesdk.CheckoutSession
		err  error
	)
	err = json.Unmarshal(event.Data.Raw, &sess)
	if err != nil {
		return err
	}
	return p.fulfillByCheckoutSession(ctx, sess)
}

func (p *PaymentUseCase) onCheckoutExpired(ctx context.Context, event stripesdk.Event) error {
	var (
		sess     stripesdk.CheckoutSession
		purchase *entity.Purchase
		err      error
	)
	err = json.Unmarshal(event.Data.Raw, &sess)
	if err != nil {
		return err
	}
	purchase, err = p.Purchases.FindBySessionID(ctx, sess.ID)
	if err != nil || purchase == nil || purchase.Status != entity.PurchaseStatusPending {
		return err
	}
	return p.markPurchaseFailed(ctx, purchase, "checkout session expired")
}

func (p *PaymentUseCase) onPaymentIntentSucceeded(ctx context.Context, event stripesdk.Event) error {
	var (
		pi stripesdk.PaymentIntent
		err error
	)
	err = json.Unmarshal(event.Data.Raw, &pi)
	if err != nil {
		return err
	}
	if pi.Metadata[stripe.MetaBillingMode] != string(entity.BillingModeAuto) {
		return nil
	}
	return p.fulfillByPaymentIntent(ctx, pi)
}

func (p *PaymentUseCase) onInvoicePaid(ctx context.Context, event stripesdk.Event) error {
	var (
		inv stripesdk.Invoice
		err error
	)
	err = json.Unmarshal(event.Data.Raw, &inv)
	if err != nil {
		return err
	}
	var (
		subDetails *stripesdk.InvoiceParentSubscriptionDetails
	)
	if inv.Parent != nil {
		subDetails = inv.Parent.SubscriptionDetails
	}
	if inv.Metadata[stripe.MetaPurchaseID] == "" && (subDetails == nil || subDetails.Metadata == nil) {
		return nil
	}
	var purchaseIDStr string
	if inv.Metadata[stripe.MetaPurchaseID] != "" {
		purchaseIDStr = inv.Metadata[stripe.MetaPurchaseID]
	} else {
		purchaseIDStr = subDetails.Metadata[stripe.MetaPurchaseID]
	}
	var (
		start = time.Unix(inv.PeriodStart, 0).UTC()
		end   = time.Unix(inv.PeriodEnd, 0).UTC()
		cust  *string
		sub   *string
		invID *string
	)
	if inv.Customer != nil {
		cust = stripeString(inv.Customer.ID)
	}
	if subDetails != nil && subDetails.Subscription != nil {
		sub = stripeString(subDetails.Subscription.ID)
	}
	invID = stripeString(inv.ID)
	return p.fulfillByPurchaseID(ctx, uuid.MustParse(purchaseIDStr), start, end, cust, sub, invID)
}

func (p *PaymentUseCase) onAccountUpdated(ctx context.Context, event stripesdk.Event) error {
	var (
		acct stripesdk.Account
		err  error
	)
	if p.ConnectUC == nil {
		return nil
	}
	err = json.Unmarshal(event.Data.Raw, &acct)
	if err != nil {
		return err
	}
	_, err = p.ConnectUC.SyncFromStripeAccountID(ctx, acct.ID)
	return err
}

func (p *PaymentUseCase) fulfillByCheckoutSession(ctx context.Context, sess stripesdk.CheckoutSession) error {
	var (
		purchase *entity.Purchase
		planID   uuid.UUID
		err      error
	)
	purchase, err = p.Purchases.FindBySessionID(ctx, sess.ID)
	if err != nil || purchase == nil {
		return err
	}
	planID, err = uuid.Parse(sess.Metadata[stripe.MetaPlanID])
	if err != nil {
		planID = uuid.Nil
	}
	return p.fulfillPurchase(ctx, purchase, planID, defaultPeriod(nil, nil), nil, nil, nil)
}

func (p *PaymentUseCase) fulfillByPaymentIntent(ctx context.Context, pi stripesdk.PaymentIntent) error {
	var (
		planID uuid.UUID
		err    error
	)
	planID, err = uuid.Parse(pi.Metadata[stripe.MetaPlanID])
	if err != nil {
		planID = uuid.Nil
	}
	return p.fulfillByPaymentIntentID(ctx, pi.ID, time.Time{}, time.Time{}, nil, nil, nil, planID)
}

func (p *PaymentUseCase) fulfillByPaymentIntentID(
	ctx context.Context,
	intentID string,
	start, end time.Time,
	customerID, subscriptionID, invoiceID *string,
	planID uuid.UUID,
) error {
	var (
		purchase *entity.Purchase
		err      error
	)
	purchase, err = p.Purchases.FindByIntentID(ctx, intentID)
	if err != nil || purchase == nil {
		return err
	}
	period := defaultPeriod(nil, nil)
	if !start.IsZero() && !end.IsZero() {
		period = periodRange{start, end}
	}
	return p.fulfillPurchase(ctx, purchase, planID, period, customerID, subscriptionID, invoiceID)
}

func (p *PaymentUseCase) fulfillByPurchaseID(ctx context.Context, purchaseID uuid.UUID, start, end time.Time, customerID, subscriptionID, invoiceID *string) error {
	var (
		purchase *entity.Purchase
		err      error
	)
	purchase, err = p.Purchases.FindByID(ctx, purchaseID)
	if err != nil || purchase == nil {
		return err
	}
	return p.fulfillPurchase(ctx, purchase, uuid.Nil, periodRange{start, end}, customerID, subscriptionID, invoiceID)
}

type periodRange struct {
	start time.Time
	end   time.Time
}

func defaultPeriod(start, end *time.Time) periodRange {
	var now time.Time
	now = time.Now().UTC()
	if start != nil && end != nil {
		return periodRange{*start, *end}
	}
	return periodRange{now, now.AddDate(0, 0, supportPeriodDays)}
}

func (p *PaymentUseCase) fulfillPurchase(
	ctx context.Context,
	purchase *entity.Purchase,
	planID uuid.UUID,
	period periodRange,
	customerID, subscriptionID, invoiceID *string,
) error {
	if purchase.Status == entity.PurchaseStatusPaid {
		return nil
	}
	if purchase.Status != entity.PurchaseStatusPending {
		return nil
	}
	return repository.WithTx(ctx, p.DB, func(txCtx context.Context) error {
		var (
			active    *entity.ProjectSupporter
			supporter entity.ProjectSupporter
			resolved  uuid.UUID
			proj      *entity.Project
			plans     []entity.ProjectPlan
			now       time.Time
			err       error
		)
		now = time.Now().UTC()
		active, err = p.Supporters.FindActive(txCtx, purchase.ProjectID, purchase.BuyerID)
		if err != nil {
			return err
		}
		if active != nil {
			return p.markPurchasePaid(txCtx, purchase, active.ID, period, invoiceID)
		}
		proj, plans, _, err = p.Projects.FindByID(txCtx, purchase.ProjectID)
		if err != nil || proj == nil {
			return ErrNotFound
		}
		resolved = planID
		if resolved == uuid.Nil {
			resolved = findPlanForPurchase(plans, purchase)
		}
		if resolved == uuid.Nil {
			return ErrNotFound
		}
		supporter = entity.ProjectSupporter{
			ID:                   uuid.New(),
			ProjectID:            purchase.ProjectID,
			UserID:               purchase.BuyerID,
			PlanID:               resolved,
			Status:               entity.SupporterStatusActive,
			BillingMode:          billingModeForPurchase(purchase),
			CurrentPeriodStart:   period.start,
			CurrentPeriodEnd:     period.end,
			StripeCustomerID:     customerID,
			StripeSubscriptionID: subscriptionID,
			CreatedAt:            now,
			UpdatedAt:            now,
		}
		if err = p.Supporters.Create(txCtx, &supporter); err != nil {
			return err
		}
		if err = p.Projects.IncrementSupportCounts(txCtx, purchase.ProjectID, resolved); err != nil {
			return err
		}
		return p.markPurchasePaid(txCtx, purchase, supporter.ID, period, invoiceID)
	})
}

func billingModeForPurchase(purchase *entity.Purchase) entity.BillingMode {
	if purchase.PaymentMethod == entity.PaymentMethodKonbini {
		return entity.BillingModeKonbini
	}
	return entity.BillingModeAuto
}

func findPlanForPurchase(plans []entity.ProjectPlan, purchase *entity.Purchase) uuid.UUID {
	if len(plans) == 1 {
		return plans[0].ID
	}
	for _, pl := range plans {
		if pl.Price == purchase.TotalPrice {
			return pl.ID
		}
	}
	if len(plans) > 0 {
		return plans[0].ID
	}
	return uuid.Nil
}

func stripeString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (p *PaymentUseCase) markPurchasePaid(ctx context.Context, purchase *entity.Purchase, supporterID uuid.UUID, period periodRange, invoiceID *string) error {
	var (
		from entity.PurchaseStatus
		err  error
	)
	from = purchase.Status
	purchase.Status = entity.PurchaseStatusPaid
	purchase.SupporterID = &supporterID
	purchase.PeriodStart = &period.start
	purchase.PeriodEnd = &period.end
	purchase.StripeInvoiceID = invoiceID
	purchase.UpdatedAt = time.Now().UTC()
	if err = p.Purchases.Update(ctx, purchase); err != nil {
		return err
	}
	return p.Purchases.RecordEvent(ctx, &entity.PurchaseEvent{
		ID:         uuid.New(),
		PurchaseID: purchase.ID,
		FromStatus: &from,
		ToStatus:   entity.PurchaseStatusPaid,
		CreatedAt:  time.Now().UTC(),
	})
}

func (p *PaymentUseCase) markPurchaseFailed(ctx context.Context, purchase *entity.Purchase, reason string) error {
	var (
		from entity.PurchaseStatus
		err  error
	)
	from = purchase.Status
	purchase.Status = entity.PurchaseStatusFailed
	purchase.UpdatedAt = time.Now().UTC()
	if err = p.Purchases.Update(ctx, purchase); err != nil {
		return err
	}
	return p.Purchases.RecordEvent(ctx, &entity.PurchaseEvent{
		ID:         uuid.New(),
		PurchaseID: purchase.ID,
		FromStatus: &from,
		ToStatus:   entity.PurchaseStatusFailed,
		Reason:     &reason,
		CreatedAt:  time.Now().UTC(),
	})
}
