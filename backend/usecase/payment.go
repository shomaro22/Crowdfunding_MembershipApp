package usecase

import (
	"context"
	"time"

	"crowdfund-app/events"
	"crowdfund-app/model"
	"crowdfund-app/repository"
	"crowdfund-app/stripe"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const supportPeriodDays = 30

type PaymentUseCase struct {
	DB              *gorm.DB
	Users           repository.IUserRepo
	Projects        repository.IProjectRepo
	Purchases       repository.IPurchaseRepo
	Supporters      repository.ISupporterRepo
	Connect         repository.IConnectRepo
	Webhooks        repository.IWebhookEventRepo
	Activity        repository.IActivityRepo
	Stripe          stripe.Client
	ConnectSync     ConnectSyncer
	SupporterViewer SupporterViewer
	PlanUpgrades    PlanUpgradeConfirmer
	supportEvents   *events.Bus
}

type CheckoutIn struct {
	BuyerID            uuid.UUID
	ProjectID          uuid.UUID
	PlanID             uuid.UUID
	PaymentMethod      string
	SuccessURL         *string
	CancelURL          *string
	FallbackSuccessURL string
	FallbackCancelURL  string
	Meta               RequestMeta
}

type CheckoutOut struct {
	SessionID string
	URL       string
}

type IntentIn struct {
	BuyerID   uuid.UUID
	ProjectID uuid.UUID
	PlanID    uuid.UUID
	Meta      RequestMeta
}

type IntentOut struct {
	ClientSecret    string
	PaymentIntentID string
}

type supportContext struct {
	buyer         model.User
	project       model.Project
	plan          model.ProjectPlan
	connect       model.ConnectAccount
	membership    model.ProjectSupporter
	hasMembership bool
	purpose       checkoutPurpose
	now           time.Time
}

type checkoutPurpose string

const (
	checkoutFirstSupport   checkoutPurpose = stripe.PurposeFirstSupport
	checkoutKonbiniRenewal checkoutPurpose = stripe.PurposeKonbiniRenewal
)

type periodRange struct {
	start time.Time
	end   time.Time
}

func (p *PaymentUseCase) logPurchaseCreated(ctx context.Context, purchase model.Purchase, meta RequestMeta) {
	uid := &purchase.BuyerID
	LogActivity(ctx, p.Activity, uid, model.PurchaseCreated, "purchase", purchase.ID.String(), meta, purchaseActivityMeta(purchase))
}

func (p *PaymentUseCase) logPaymentCheckoutStarted(ctx context.Context, purchase model.Purchase, meta RequestMeta, flow string) {
	uid := &purchase.BuyerID
	fields := map[string]string{
		"flow":       flow,
		"project_id": purchase.ProjectID.String(),
	}
	LogActivity(ctx, p.Activity, uid, model.PaymentCheckout, "purchase", purchase.ID.String(), meta, activityMeta(fields))
}

func (p *PaymentUseCase) logPurchaseCancelled(ctx context.Context, purchase model.Purchase, reason string, to model.PurchaseStatus) {
	uid := &purchase.BuyerID
	LogActivity(ctx, p.Activity, uid, model.PurchaseCancelled, "purchase", purchase.ID.String(), RequestMeta{}, activityMeta(map[string]string{
		"reason":    reason,
		"to_status": string(to),
	}))
}

func (p *PaymentUseCase) logPaymentWebhook(ctx context.Context, eventType, eventID string) {
	LogActivity(ctx, p.Activity, nil, model.PaymentWebhook, "stripe_event", eventID, RequestMeta{}, activityMeta(map[string]string{
		"event_type": eventType,
	}))
}

func (p *PaymentUseCase) newPendingPurchase(
	sc supportContext,
	pm model.PaymentMethod,
	checkoutID, intentID *string,
) model.Purchase {
	var (
		row model.Purchase
		key string
	)
	key = uuid.New().String()
	row = model.Purchase{
		ID:               uuid.New(),
		ProjectID:        sc.project.ID,
		BuyerID:          sc.buyer.ID,
		Quantity:         1,
		TotalPrice:       sc.plan.Price,
		Status:           model.PurchaseStatusPending,
		PaymentMethod:    pm,
		StripeCheckoutID: checkoutID,
		StripeIntentID:   intentID,
		IdempotencyKey:   &key,
		CreatedAt:        sc.now,
		UpdatedAt:        sc.now,
	}
	if sc.purpose == checkoutKonbiniRenewal && sc.hasMembership {
		row.SupporterID = &sc.membership.ID
	}
	return row
}

func (p *PaymentUseCase) savePendingAfterStripe(ctx context.Context, purchase model.Purchase) error {
	return repository.WithTx(ctx, p.DB, func(txCtx context.Context) error {
		return p.Purchases.Create(txCtx, &purchase)
	})
}

func checkoutMetadata(sc supportContext, purchaseID uuid.UUID, pm model.PaymentMethod) map[string]string {
	var mode model.BillingMode
	if pm == model.PaymentMethodKonbini {
		mode = model.BillingModeKonbini
	} else {
		mode = model.BillingModeAuto
	}
	meta := paymentMetadata(
		purchaseID,
		sc.project.ID,
		sc.plan.ID,
		sc.buyer.ID,
		mode,
	)
	meta[stripe.MetaCheckoutPurpose] = string(sc.purpose)
	if sc.hasMembership {
		meta[stripe.MetaSupporterID] = sc.membership.ID.String()
	}
	return meta
}

func paymentMetadata(purchaseID, projectID, planID, buyerID uuid.UUID, mode model.BillingMode) map[string]string {
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

func parseCheckoutPurpose(meta map[string]string) checkoutPurpose {
	if meta == nil {
		return checkoutFirstSupport
	}
	switch meta[stripe.MetaCheckoutPurpose] {
	case stripe.PurposeKonbiniRenewal:
		return checkoutKonbiniRenewal
	default:
		return checkoutFirstSupport
	}
}

func isKonbiniPurchase(purchase model.Purchase) bool {
	return purchase.PaymentMethod == model.PaymentMethodKonbini
}

func (p *PaymentUseCase) CreateCheckout(ctx context.Context, in CheckoutIn) (CheckoutOut, error) {
	var (
		sc       supportContext
		purchase model.Purchase
		meta     map[string]string
		sess     stripe.CheckoutSessionOut
		pm       model.PaymentMethod
		err      error
	)
	if p.Stripe == nil {
		return CheckoutOut{}, ErrStripeUnavailable
	}
	pm = model.PaymentMethod(in.PaymentMethod)
	if pm == "" {
		pm = model.PaymentMethodKonbini
	}
	if pm != model.PaymentMethodKonbini && pm != model.PaymentMethodCard {
		return CheckoutOut{}, ErrBadRequest
	}
	sc, err = p.loadCheckoutContext(ctx, in.BuyerID, in.ProjectID, in.PlanID, pm)
	if err != nil {
		return CheckoutOut{}, err
	}
	purchase = p.newPendingPurchase(sc, pm, nil, nil)
	meta = checkoutMetadata(sc, purchase.ID, pm)
	sess, err = p.Stripe.CreateCheckoutSession(ctx, stripe.CheckoutSessionIn{
		ProductName:      sc.project.Title,
		Amount:           int64(sc.plan.Price),
		ConnectAccountID: sc.connect.StripeAccountID,
		CustomerEmail:    sc.buyer.Email,
		SuccessURL:       checkoutURL(in.SuccessURL, in.FallbackSuccessURL),
		CancelURL:        checkoutURL(in.CancelURL, in.FallbackCancelURL),
		PaymentMethod:    string(pm),
		Metadata:         meta,
	})
	if err != nil {
		return CheckoutOut{}, err
	}
	purchase.StripeCheckoutID = &sess.SessionID
	if sess.PaymentIntentID != "" {
		purchase.StripeIntentID = &sess.PaymentIntentID
	}
	purchase.UpdatedAt = sc.now
	if err = p.savePendingAfterStripe(ctx, purchase); err != nil {
		return CheckoutOut{}, err
	}
	p.logPaymentCheckoutStarted(ctx, purchase, in.Meta, "checkout")
	p.logPurchaseCreated(ctx, purchase, in.Meta)
	return CheckoutOut{SessionID: sess.SessionID, URL: sess.URL}, nil
}

func (p *PaymentUseCase) CreateIntent(ctx context.Context, in IntentIn) (IntentOut, error) {
	var (
		sc       supportContext
		purchase model.Purchase
		meta     map[string]string
		sub      stripe.SubscriptionIntentOut
		err      error
	)
	if p.Stripe == nil {
		return IntentOut{}, ErrStripeUnavailable
	}
	sc, err = p.loadSupportContext(ctx, in.BuyerID, in.ProjectID, in.PlanID)
	if err != nil {
		return IntentOut{}, err
	}
	purchase = p.newPendingPurchase(sc, model.PaymentMethodCard, nil, nil)
	meta = paymentMetadata(purchase.ID, in.ProjectID, in.PlanID, in.BuyerID, model.BillingModeAuto)
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
	if err = p.savePendingAfterStripe(ctx, purchase); err != nil {
		return IntentOut{}, err
	}
	p.logPaymentCheckoutStarted(ctx, purchase, in.Meta, "intent")
	p.logPurchaseCreated(ctx, purchase, in.Meta)
	return IntentOut{ClientSecret: sub.ClientSecret, PaymentIntentID: sub.PaymentIntentID}, nil
}

func (p *PaymentUseCase) VerifyCheckout(ctx context.Context, buyerID uuid.UUID, sessionID string) (SupporterView, error) {
	var (
		sess     stripe.CheckoutSessionSnapshot
		purchase model.Purchase
		err      error
	)
	if p.Stripe == nil {
		return SupporterView{}, ErrStripeUnavailable
	}
	if sessionID == "" {
		return SupporterView{}, ErrBadRequest
	}
	sess, err = p.Stripe.GetCheckoutSession(ctx, sessionID)
	if err != nil {
		return SupporterView{}, ErrBadRequest
	}
	purchase, err = p.Purchases.GetSessionID(ctx, sessionID)
	if err != nil {
		return SupporterView{}, mapRepoNotFound(err)
	}
	return p.verifyAndFulfill(ctx, buyerID, purchase, sess.PaymentStatus == stripe.CheckoutPaymentStatusPaid, func(ctx context.Context) error {
		return p.fulfillCheckoutSession(ctx, sess)
	})
}

func (p *PaymentUseCase) VerifyIntent(ctx context.Context, buyerID uuid.UUID, intentID string) (SupporterView, error) {
	var (
		pi       stripe.PaymentIntentDetail
		purchase model.Purchase
		err      error
	)
	if p.Stripe == nil {
		return SupporterView{}, ErrStripeUnavailable
	}
	if intentID == "" {
		return SupporterView{}, ErrBadRequest
	}
	pi, err = p.Stripe.GetPaymentIntentDetail(ctx, intentID)
	if err != nil {
		return SupporterView{}, ErrBadRequest
	}
	if pi.Metadata[stripe.MetaCheckoutPurpose] == stripe.PurposePlanUpgrade {
		if p.PlanUpgrades == nil {
			return SupporterView{}, ErrNotFound
		}
		purchase, err = p.Purchases.GetIntentID(ctx, intentID)
		if err != nil {
			return SupporterView{}, mapRepoNotFound(err)
		}
		if err = verifyPurchaseBuyer(purchase, buyerID); err != nil {
			return SupporterView{}, err
		}
		if purchase.SupporterID == nil {
			return SupporterView{}, ErrBadRequest
		}
		if pi.Status != stripe.PaymentIntentStatusSucceeded {
			return SupporterView{}, ErrBadRequest
		}
		return p.PlanUpgrades.UpgradeConfirm(ctx, buyerID, *purchase.SupporterID, intentID)
	}
	purchase, err = p.Purchases.GetIntentID(ctx, intentID)
	if err != nil {
		return SupporterView{}, mapRepoNotFound(err)
	}
	return p.verifyAndFulfill(ctx, buyerID, purchase, pi.Status == stripe.PaymentIntentStatusSucceeded, func(ctx context.Context) error {
		return p.fulfillPaymentIntent(ctx, pi)
	})
}

func (p *PaymentUseCase) verifyAndFulfill(
	ctx context.Context,
	buyerID uuid.UUID,
	purchase model.Purchase,
	paid bool,
	fulfill func(context.Context) error,
) (SupporterView, error) {
	var (
		refreshed model.Purchase
		err       error
	)
	if err = verifyPurchaseBuyer(purchase, buyerID); err != nil {
		return SupporterView{}, err
	}
	if purchase.Status == model.PurchaseStatusPaid {
		return p.supporterViewForPurchase(ctx, purchase)
	}
	if !paid {
		return SupporterView{}, ErrBadRequest
	}
	if err = fulfill(ctx); err != nil {
		return SupporterView{}, err
	}
	refreshed, err = p.Purchases.GetID(ctx, purchase.ID)
	if err != nil {
		return SupporterView{}, mapRepoNotFound(err)
	}
	return p.supporterViewForPurchase(ctx, refreshed)
}

func verifyPurchaseBuyer(purchase model.Purchase, buyerID uuid.UUID) error {
	if purchase.BuyerID != buyerID {
		return ErrForbidden
	}
	return nil
}

func (p *PaymentUseCase) supporterViewForPurchase(ctx context.Context, purchase model.Purchase) (SupporterView, error) {
	var (
		supporter model.ProjectSupporter
		err       error
	)
	if purchase.SupporterID != nil {
		supporter, err = p.Supporters.GetID(ctx, *purchase.SupporterID)
	} else {
		supporter, err = p.Supporters.GetActive(ctx, purchase.ProjectID, purchase.BuyerID)
	}
	if err != nil {
		return SupporterView{}, mapRepoNotFound(err)
	}
	if p.SupporterViewer == nil {
		return SupporterView{}, ErrNotFound
	}
	return p.SupporterViewer.ViewForSupporter(ctx, supporter)
}

const konbiniRenewalWindowDays = 7

func (p *PaymentUseCase) loadSupportContext(ctx context.Context, buyerID, projectID, planID uuid.UUID) (supportContext, error) {
	var (
		sc  supportContext
		err error
	)
	sc, err = p.loadCheckoutBase(ctx, buyerID, projectID, planID)
	if err != nil {
		return sc, err
	}
	sc.membership, err = p.Supporters.GetMembership(ctx, projectID, buyerID)
	if isRepoNotFound(err) {
		sc.hasMembership = false
	} else if err != nil {
		return sc, err
	} else {
		sc.hasMembership = true
	}
	if sc.hasMembership {
		return sc, ErrAlreadySupporting
	}
	if err = checkPlanCapacity(sc.plan); err != nil {
		return sc, err
	}
	sc.purpose = checkoutFirstSupport
	return sc, nil
}

func (p *PaymentUseCase) loadCheckoutContext(ctx context.Context, buyerID, projectID, planID uuid.UUID, pm model.PaymentMethod) (supportContext, error) {
	if pm == model.PaymentMethodKonbini {
		return p.loadKonbiniCheckoutContext(ctx, buyerID, projectID, planID)
	}
	return p.loadSupportContext(ctx, buyerID, projectID, planID)
}

func (p *PaymentUseCase) loadKonbiniCheckoutContext(ctx context.Context, buyerID, projectID, planID uuid.UUID) (supportContext, error) {
	var (
		sc  supportContext
		err error
	)
	sc, err = p.loadCheckoutBase(ctx, buyerID, projectID, planID)
	if err != nil {
		return sc, err
	}
	sc.membership, err = p.Supporters.GetMembership(ctx, projectID, buyerID)
	if isRepoNotFound(err) {
		sc.hasMembership = false
	} else if err != nil {
		return sc, err
	} else {
		sc.hasMembership = true
	}
	if !sc.hasMembership {
		if err = checkPlanCapacity(sc.plan); err != nil {
			return sc, err
		}
		sc.purpose = checkoutFirstSupport
		return sc, nil
	}
	if sc.membership.BillingMode != model.BillingModeKonbini {
		return sc, ErrBadRequest
	}
	if !canIssueKonbiniRenewal(sc.membership, sc.now) {
		return sc, ErrBadRequest
	}
	sc.purpose = checkoutKonbiniRenewal
	return sc, nil
}

func (p *PaymentUseCase) loadCheckoutBase(ctx context.Context, buyerID, projectID, planID uuid.UUID) (supportContext, error) {
	var (
		sc   supportContext
		pend model.Purchase
		err  error
	)
	sc, err = p.loadBuyerProjectPlan(ctx, buyerID, projectID, planID)
	if err != nil {
		return sc, err
	}
	pend, err = p.Purchases.GetPendingBuyerProject(ctx, buyerID, projectID)
	if isRepoNotFound(err) {
		// no pending checkout
	} else if err != nil {
		return sc, err
	} else if pend.ID != uuid.Nil {
		pend.Status = model.PurchaseStatusCancelled
		pend.UpdatedAt = sc.now
		if err = p.Purchases.Update(ctx, &pend); err != nil {
			return sc, err
		}
	}
	return sc, nil
}

func checkPlanCapacity(plan model.ProjectPlan) error {
	if plan.MaxSupporters != nil && plan.SupporterCount >= *plan.MaxSupporters {
		return ErrForbidden
	}
	return nil
}

func (p *PaymentUseCase) loadBuyerProjectPlan(ctx context.Context, buyerID, projectID, planID uuid.UUID) (supportContext, error) {
	var (
		sc        supportContext
		u         model.User
		project   model.Project
		plans     []model.ProjectPlan
		plan      model.ProjectPlan
		planFound bool
		conn      model.ConnectAccount
		i         int
		err       error
	)
	sc.now = time.Now().UTC()
	u, err = p.Users.GetID(ctx, buyerID)
	if err != nil {
		return sc, mapRepoNotFound(err)
	}
	if !u.IsActive() || u.Role != model.UserRoleBuyer {
		return sc, ErrForbidden
	}
	project, plans, _, err = p.Projects.GetID(ctx, projectID)
	if err != nil {
		return sc, mapRepoNotFound(err)
	}
	if !project.IsPublic(sc.now) {
		return sc, ErrForbidden
	}
	for i = range plans {
		if plans[i].ID == planID {
			if !plans[i].IsActive {
				return sc, ErrNotFound
			}
			plan = plans[i]
			planFound = true
			break
		}
	}
	if !planFound {
		return sc, ErrNotFound
	}
	conn, err = p.Connect.GetUserID(ctx, project.SellerID)
	if isRepoNotFound(err) {
		return sc, ErrForbidden
	}
	if err != nil {
		return sc, err
	}
	if !conn.CanCharge() {
		return sc, ErrForbidden
	}
	sc.buyer = u
	sc.project = project
	sc.plan = plan
	sc.connect = conn
	return sc, nil
}

func canIssueKonbiniRenewal(s model.ProjectSupporter, now time.Time) bool {
	var windowStart time.Time
	if s.Status == model.SupporterStatusPendingRenewal {
		return true
	}
	if s.Status != model.SupporterStatusActive || s.BillingMode != model.BillingModeKonbini {
		return false
	}
	windowStart = s.CurrentPeriodEnd.AddDate(0, 0, -konbiniRenewalWindowDays)
	return !now.Before(windowStart)
}

func (p *PaymentUseCase) fulfillPaymentIntentID(
	ctx context.Context,
	intentID string,
	start, end time.Time,
	customerID, subscriptionID, invoiceID *string,
	planID uuid.UUID,
) error {
	var (
		purchase model.Purchase
		err      error
	)
	purchase, err = p.Purchases.GetIntentID(ctx, intentID)
	if isRepoNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	period := defaultPeriod(nil, nil)
	if !start.IsZero() && !end.IsZero() {
		period = periodRange{start, end}
	}
	return p.fulfillFirstSupport(ctx, purchase, planID, period, customerID, subscriptionID, invoiceID)
}

func defaultPeriod(start, end *time.Time) periodRange {
	var now time.Time
	now = time.Now().UTC()
	if start != nil && end != nil {
		return periodRange{*start, *end}
	}
	return periodRange{now, now.AddDate(0, 0, supportPeriodDays)}
}

func (p *PaymentUseCase) fulfillFirstSupport(
	ctx context.Context,
	purchase model.Purchase,
	planID uuid.UUID,
	period periodRange,
	customerID, subscriptionID, invoiceID *string,
) error {
	var (
		completed *events.SupportCompleted
		err       error
	)

	err = repository.WithTx(ctx, p.DB, func(txCtx context.Context) error {
		var (
			membership model.ProjectSupporter
			hasMember  bool
			supporter  model.ProjectSupporter
			resolved   uuid.UUID
			plans      []model.ProjectPlan
			now        time.Time
			txErr      error
		)
		purchase, txErr = p.Purchases.GetIDLock(txCtx, purchase.ID)
		if txErr != nil {
			return mapRepoNotFound(txErr)
		}
		if purchase.Status == model.PurchaseStatusPaid {
			return nil
		}
		if purchase.Status != model.PurchaseStatusPending {
			return nil
		}
		now = time.Now().UTC()
		membership, txErr = p.Supporters.GetMembership(txCtx, purchase.ProjectID, purchase.BuyerID)
		if isRepoNotFound(txErr) {
			hasMember = false
		} else if txErr != nil {
			return txErr
		} else {
			hasMember = true
		}
		if hasMember {
			completed = &events.SupportCompleted{
				PurchaseID:  purchase.ID,
				ProjectID:   purchase.ProjectID,
				BuyerID:     purchase.BuyerID,
				SupporterID: membership.ID,
				PlanID:      membership.PlanID,
				Amount:      int64(purchase.TotalPrice),
				PeriodStart: period.start,
				PeriodEnd:   period.end,
			}
			return p.markPurchasePaid(txCtx, purchase, membership.ID, period, invoiceID)
		}
		_, plans, _, txErr = p.Projects.GetID(txCtx, purchase.ProjectID)
		if txErr != nil {
			return mapRepoNotFound(txErr)
		}
		resolved = planID
		if resolved == uuid.Nil {
			resolved = findPlanForPurchase(plans, purchase)
		}
		if resolved == uuid.Nil {
			return ErrNotFound
		}
		supporter = model.ProjectSupporter{
			ID:                   uuid.New(),
			ProjectID:            purchase.ProjectID,
			UserID:               purchase.BuyerID,
			PlanID:               resolved,
			Status:               model.SupporterStatusActive,
			BillingMode:          billingModeForPurchase(purchase),
			CurrentPeriodStart:   period.start,
			CurrentPeriodEnd:     period.end,
			StripeCustomerID:     customerID,
			StripeSubscriptionID: subscriptionID,
			CreatedAt:            now,
			UpdatedAt:            now,
		}
		if txErr = p.Supporters.Create(txCtx, &supporter); txErr != nil {
			return txErr
		}
		if txErr = p.Projects.IncrementSupportCounts(txCtx, purchase.ProjectID, resolved); txErr != nil {
			return txErr
		}
		completed = &events.SupportCompleted{
			PurchaseID:  purchase.ID,
			ProjectID:   purchase.ProjectID,
			BuyerID:     purchase.BuyerID,
			SupporterID: supporter.ID,
			PlanID:      resolved,
			Amount:      int64(purchase.TotalPrice),
			PeriodStart: period.start,
			PeriodEnd:   period.end,
		}
		return p.markPurchasePaid(txCtx, purchase, supporter.ID, period, invoiceID)
	})
	if err != nil {
		return err
	}
	if completed != nil && p.supportEvents != nil {
		return p.supportEvents.Publish(ctx, *completed)
	}
	return nil
}

func billingModeForPurchase(purchase model.Purchase) model.BillingMode {
	if purchase.PaymentMethod == model.PaymentMethodKonbini {
		return model.BillingModeKonbini
	}
	return model.BillingModeAuto
}

func findPlanForPurchase(plans []model.ProjectPlan, purchase model.Purchase) uuid.UUID {
	if len(plans) == 1 {
		return plans[0].ID
	}
	for _, plan := range plans {
		if plan.Price == purchase.TotalPrice {
			return plan.ID
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

func (p *PaymentUseCase) markPurchasePaid(ctx context.Context, purchase model.Purchase, supporterID uuid.UUID, period periodRange, invoiceID *string) error {
	var (
		from model.PurchaseStatus
		err  error
	)
	from = purchase.Status
	purchase.Status = model.PurchaseStatusPaid
	purchase.SupporterID = &supporterID
	purchase.PeriodStart = &period.start
	purchase.PeriodEnd = &period.end
	purchase.StripeInvoiceID = invoiceID
	purchase.UpdatedAt = time.Now().UTC()
	if err = p.Purchases.Update(ctx, &purchase); err != nil {
		return err
	}
	if err = p.Projects.AddCurrentAmount(ctx, purchase.ProjectID, int64(purchase.TotalPrice)); err != nil {
		return err
	}
	return p.Purchases.RecordEvent(ctx, &model.PurchaseEvent{
		ID:         uuid.New(),
		PurchaseID: purchase.ID,
		FromStatus: &from,
		ToStatus:   model.PurchaseStatusPaid,
		CreatedAt:  time.Now().UTC(),
	})
}

func (p *PaymentUseCase) markPurchaseFailed(ctx context.Context, purchase model.Purchase, reason string) error {
	return p.markPurchaseTerminal(ctx, purchase, model.PurchaseStatusFailed, reason)
}

func (p *PaymentUseCase) markPurchaseCancelled(ctx context.Context, purchase model.Purchase, reason string) error {
	return p.markPurchaseTerminal(ctx, purchase, model.PurchaseStatusCancelled, reason)
}

func (p *PaymentUseCase) markPurchaseTerminal(ctx context.Context, purchase model.Purchase, to model.PurchaseStatus, reason string) error {
	var (
		from model.PurchaseStatus
		err  error
	)
	from = purchase.Status
	purchase.Status = to
	purchase.UpdatedAt = time.Now().UTC()
	if err = p.Purchases.Update(ctx, &purchase); err != nil {
		return err
	}
	if err = p.Purchases.RecordEvent(ctx, &model.PurchaseEvent{
		ID:         uuid.New(),
		PurchaseID: purchase.ID,
		FromStatus: &from,
		ToStatus:   to,
		Reason:     &reason,
		CreatedAt:  time.Now().UTC(),
	}); err != nil {
		return err
	}
	if to == model.PurchaseStatusCancelled || to == model.PurchaseStatusFailed {
		p.logPurchaseCancelled(ctx, purchase, reason, to)
	}
	return nil
}

func (p *PaymentUseCase) fulfillCheckoutSession(ctx context.Context, sess stripe.CheckoutSessionSnapshot) error {
	var (
		purchase model.Purchase
		planID   uuid.UUID
		purpose  checkoutPurpose
		err      error
	)
	purchase, err = p.Purchases.GetSessionID(ctx, sess.ID)
	if isRepoNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if purchase.Status == model.PurchaseStatusPaid {
		return nil
	}
	if purchase.Status != model.PurchaseStatusPending {
		return nil
	}
	planID, err = uuid.Parse(sess.Metadata[stripe.MetaPlanID])
	if err != nil {
		planID = uuid.Nil
	}
	purpose = parseCheckoutPurpose(sess.Metadata)
	if purpose == checkoutKonbiniRenewal {
		return p.fulfillKonbiniRenewal(ctx, purchase, planID, time.Now().UTC())
	}
	if isKonbiniPurchase(purchase) {
		return p.fulfillKonbiniFirstSupport(ctx, purchase, planID, time.Now().UTC())
	}
	return p.fulfillFirstSupport(ctx, purchase, planID, defaultPeriod(nil, nil), nil, nil, nil)
}

func (p *PaymentUseCase) fulfillKonbiniFirstSupport(ctx context.Context, purchase model.Purchase, planID uuid.UUID, paidAt time.Time) error {
	return p.fulfillFirstSupport(ctx, purchase, planID, konbiniFirstPeriod(paidAt), nil, nil, nil)
}

func (p *PaymentUseCase) fulfillKonbiniRenewal(ctx context.Context, purchase model.Purchase, planID uuid.UUID, paidAt time.Time) error {
	return repository.WithTx(ctx, p.DB, func(txCtx context.Context) error {
		var (
			supporter model.ProjectSupporter
			period    periodRange
			now       time.Time
			err       error
		)
		purchase, err = p.Purchases.GetIDLock(txCtx, purchase.ID)
		if err != nil {
			return mapRepoNotFound(err)
		}
		if purchase.Status == model.PurchaseStatusPaid {
			return nil
		}
		if purchase.Status != model.PurchaseStatusPending {
			return nil
		}
		if purchase.SupporterID == nil {
			return ErrNotFound
		}
		supporter, err = p.Supporters.GetID(txCtx, *purchase.SupporterID)
		if err != nil {
			return mapRepoNotFound(err)
		}
		period = konbiniRenewalPeriod(supporter, paidAt)
		now = time.Now().UTC()
		supporter.Status = model.SupporterStatusActive
		supporter.CurrentPeriodStart = period.start
		supporter.CurrentPeriodEnd = period.end
		if planID != uuid.Nil {
			supporter.PlanID = planID
		}
		supporter.UpdatedAt = now
		if err = p.Supporters.Update(txCtx, &supporter); err != nil {
			return err
		}
		return p.markPurchasePaid(txCtx, purchase, supporter.ID, period, nil)
	})
}

func (p *PaymentUseCase) handleKonbiniCheckoutUnpaid(ctx context.Context, purchase model.Purchase, reason string, cancelled bool) error {
	if purchase.Status != model.PurchaseStatusPending {
		return nil
	}
	if !isKonbiniPurchase(purchase) {
		if cancelled {
			return p.markPurchaseCancelled(ctx, purchase, reason)
		}
		return p.markPurchaseFailed(ctx, purchase, reason)
	}
	if purchase.SupporterID != nil {
		return p.failKonbiniRenewal(ctx, purchase, reason, cancelled)
	}
	if cancelled {
		return p.markPurchaseCancelled(ctx, purchase, reason)
	}
	return p.markPurchaseFailed(ctx, purchase, reason)
}

func (p *PaymentUseCase) failKonbiniRenewal(ctx context.Context, purchase model.Purchase, reason string, cancelled bool) error {
	return repository.WithTx(ctx, p.DB, func(txCtx context.Context) error {
		var (
			supporter model.ProjectSupporter
			err       error
		)
		if cancelled {
			err = p.markPurchaseCancelled(txCtx, purchase, reason)
		} else {
			err = p.markPurchaseFailed(txCtx, purchase, reason)
		}
		if err != nil {
			return err
		}
		if purchase.SupporterID == nil {
			return nil
		}
		supporter, err = p.Supporters.GetID(txCtx, *purchase.SupporterID)
		if isRepoNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if supporter.Status == model.SupporterStatusPendingRenewal {
			return nil
		}
		supporter.Status = model.SupporterStatusPendingRenewal
		supporter.UpdatedAt = time.Now().UTC()
		return p.Supporters.Update(txCtx, &supporter)
	})
}
func konbiniFirstPeriod(paidAt time.Time) periodRange {
	return periodRange{start: paidAt, end: paidAt.AddDate(0, 0, supportPeriodDays)}
}

func konbiniRenewalPeriod(supporter model.ProjectSupporter, paidAt time.Time) periodRange {
	var start time.Time
	start = supporter.CurrentPeriodEnd
	if paidAt.After(start) {
		start = paidAt
	}
	return periodRange{start: start, end: start.AddDate(0, 0, supportPeriodDays)}
}

func (p *PaymentUseCase) fulfillPaymentIntent(ctx context.Context, pi stripe.PaymentIntentDetail) error {
	var (
		planID uuid.UUID
		err    error
	)
	if pi.Metadata[stripe.MetaCheckoutPurpose] == stripe.PurposePlanUpgrade {
		return nil
	}
	if pi.Metadata[stripe.MetaBillingMode] == string(model.BillingModeAuto) {
		planID, err = uuid.Parse(pi.Metadata[stripe.MetaPlanID])
		if err != nil {
			planID = uuid.Nil
		}
		return p.fulfillPaymentIntentID(ctx, pi.ID, time.Time{}, time.Time{}, nil, nil, nil, planID)
	}
	if pi.Metadata[stripe.MetaBillingMode] != string(model.BillingModeKonbini) {
		return nil
	}
	return p.fulfillKonbiniPaymentIntent(ctx, pi)
}

func (p *PaymentUseCase) fulfillKonbiniPaymentIntent(ctx context.Context, pi stripe.PaymentIntentDetail) error {
	var (
		purchase model.Purchase
		planID   uuid.UUID
		purpose  checkoutPurpose
		err      error
	)
	purchase, err = p.Purchases.GetIntentID(ctx, pi.ID)
	if isRepoNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	planID, err = uuid.Parse(pi.Metadata[stripe.MetaPlanID])
	if err != nil {
		planID = uuid.Nil
	}
	purpose = parseCheckoutPurpose(pi.Metadata)
	if purpose == checkoutKonbiniRenewal {
		return p.fulfillKonbiniRenewal(ctx, purchase, planID, time.Now().UTC())
	}
	return p.fulfillKonbiniFirstSupport(ctx, purchase, planID, time.Now().UTC())
}
