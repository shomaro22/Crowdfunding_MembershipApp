package usecase

import (
	"context"
	"time"

	"crowdfund-app/model"
	"crowdfund-app/repository"
	"crowdfund-app/stripe"

	"github.com/google/uuid"
)

func (s *SupporterUseCase) Upgrade(ctx context.Context, userID, supporterID, planID uuid.UUID) (UpgradeOut, error) {
	var (
		supporter model.ProjectSupporter
		current   model.ProjectPlan
		target    model.ProjectPlan
		project   model.Project
		conn      model.ConnectAccount
		now       time.Time
		err       error
	)
	now = time.Now().UTC()
	supporter, err = s.loadOwned(ctx, userID, supporterID)
	if err != nil {
		return UpgradeOut{}, err
	}
	if !supporterCanManage(supporter, now) || supporter.Status == model.SupporterStatusCancelled {
		return UpgradeOut{}, ErrForbidden
	}
	if supporter.PendingPlanID != nil {
		return UpgradeOut{}, ErrBadRequest
	}
	target, err = s.Projects.GetPlan(ctx, supporter.ProjectID, planID)
	if err != nil {
		return UpgradeOut{}, mapRepoNotFound(err)
	}
	if !target.IsActive {
		return UpgradeOut{}, ErrNotFound
	}
	if target.ID == supporter.PlanID {
		return UpgradeOut{}, ErrBadRequest
	}
	current, err = s.Projects.GetPlan(ctx, supporter.ProjectID, supporter.PlanID)
	if err != nil {
		return UpgradeOut{}, mapRepoNotFound(err)
	}
	if target.MembershipType.Rank() <= current.MembershipType.Rank() {
		return s.scheduleDowngrade(ctx, supporter, target, now)
	}
	if s.Stripe == nil {
		return UpgradeOut{}, ErrStripeUnavailable
	}
	project, _, _, err = s.Projects.GetID(ctx, supporter.ProjectID)
	if err != nil {
		return UpgradeOut{}, mapRepoNotFound(err)
	}
	conn, err = s.Connect.GetUserID(ctx, project.SellerID)
	if isRepoNotFound(err) {
		return UpgradeOut{}, ErrForbidden
	}
	if err != nil {
		return UpgradeOut{}, err
	}
	if !conn.CanCharge() {
		return UpgradeOut{}, ErrForbidden
	}
	return s.startPaidUpgrade(ctx, supporter, current, target, conn, now)
}

func (s *SupporterUseCase) scheduleDowngrade(ctx context.Context, supporter model.ProjectSupporter, target model.ProjectPlan, now time.Time) (UpgradeOut, error) {
	var (
		changeAt time.Time
		out      UpgradeOut
		err      error
	)
	changeAt = supporter.CurrentPeriodEnd
	supporter.PendingPlanID = &target.ID
	supporter.PlanChangeDate = &changeAt
	supporter.UpdatedAt = now
	if err = s.Supporters.Update(ctx, &supporter); err != nil {
		return UpgradeOut{}, err
	}
	out.Supporter, err = s.enrichView(ctx, supporter)
	return out, err
}

func (s *SupporterUseCase) startPaidUpgrade(
	ctx context.Context,
	supporter model.ProjectSupporter,
	current, target model.ProjectPlan,
	conn model.ConnectAccount,
	now time.Time,
) (UpgradeOut, error) {
	var (
		diff     int64
		meta     map[string]string
		pi       stripe.UpgradePaymentOut
		purchase model.Purchase
		key      string
		client   *string
		out      UpgradeOut
		err      error
	)
	diff = int64(target.Price - current.Price)
	if diff <= 0 {
		return UpgradeOut{}, ErrBadRequest
	}
	meta = map[string]string{
		stripe.MetaSupporterID:     supporter.ID.String(),
		stripe.MetaPlanID:          target.ID.String(),
		stripe.MetaProjectID:       supporter.ProjectID.String(),
		stripe.MetaBuyerID:         supporter.UserID.String(),
		stripe.MetaCheckoutPurpose: stripe.PurposePlanUpgrade,
		stripe.MetaBillingMode:     string(supporter.BillingMode),
	}
	custID := ""
	if supporter.StripeCustomerID != nil {
		custID = *supporter.StripeCustomerID
	}
	pi, err = s.Stripe.CreateUpgradePaymentIntent(ctx, stripe.UpgradePaymentIn{
		Amount:           diff,
		ConnectAccountID: conn.StripeAccountID,
		CustomerID:       custID,
		Metadata:         meta,
	})
	if err != nil {
		return UpgradeOut{}, err
	}
	key = uuid.New().String()
	purchase = model.Purchase{
		ID:             uuid.New(),
		ProjectID:      supporter.ProjectID,
		SupporterID:    &supporter.ID,
		BuyerID:        supporter.UserID,
		Quantity:       1,
		TotalPrice:     int(diff),
		Status:         model.PurchaseStatusPending,
		PaymentMethod:  model.PaymentMethodCard,
		StripeIntentID: &pi.PaymentIntentID,
		IdempotencyKey: &key,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err = s.Purchases.Create(ctx, &purchase); err != nil {
		return UpgradeOut{}, err
	}
	LogActivity(ctx, s.Activity, &supporter.UserID, model.PurchaseCreated, "purchase", purchase.ID.String(), RequestMeta{}, activityMeta(map[string]string{
		"project_id": supporter.ProjectID.String(),
		"purpose":    "plan_upgrade",
	}))
	changeAt := now
	supporter.PendingPlanID = &target.ID
	supporter.PlanChangeDate = &changeAt
	supporter.UpdatedAt = now
	if err = s.Supporters.Update(ctx, &supporter); err != nil {
		return UpgradeOut{}, err
	}
	client = &pi.ClientSecret
	out.PaymentIntentClientSecret = client
	out.Supporter, err = s.enrichView(ctx, supporter)
	return out, err
}

func (s *SupporterUseCase) UpgradeConfirm(ctx context.Context, userID, supporterID uuid.UUID, intentID string) (SupporterView, error) {
	var (
		supporter model.ProjectSupporter
		purchase  model.Purchase
		pi        stripe.PaymentIntentSnapshot
		now       time.Time
		err       error
	)
	if s.Stripe == nil {
		return SupporterView{}, ErrStripeUnavailable
	}
	now = time.Now().UTC()
	supporter, err = s.loadOwned(ctx, userID, supporterID)
	if err != nil {
		return SupporterView{}, err
	}
	if supporter.PendingPlanID == nil {
		return SupporterView{}, ErrBadRequest
	}
	purchase, err = s.Purchases.GetIntentID(ctx, intentID)
	if err != nil {
		return SupporterView{}, mapRepoNotFound(err)
	}
	if purchase.SupporterID == nil || *purchase.SupporterID != supporter.ID {
		return SupporterView{}, ErrNotFound
	}
	pi, err = s.Stripe.GetPaymentIntent(ctx, intentID)
	if err != nil {
		return SupporterView{}, err
	}
	if pi.Status != stripe.PaymentIntentStatusSucceeded {
		return SupporterView{}, ErrBadRequest
	}
	return s.finalizePaidUpgrade(ctx, supporter, purchase, now)
}

func (s *SupporterUseCase) finalizePaidUpgrade(ctx context.Context, supporter model.ProjectSupporter, purchase model.Purchase, now time.Time) (SupporterView, error) {
	var (
		locked    model.ProjectSupporter
		newPlanID uuid.UUID
		err       error
	)
	err = repository.WithTx(ctx, s.DB, func(txCtx context.Context) error {
		locked, err = s.Supporters.GetID(txCtx, supporter.ID)
		if err != nil {
			return mapRepoNotFound(err)
		}
		if locked.PendingPlanID == nil {
			return ErrNotFound
		}
		newPlanID = *locked.PendingPlanID
		if purchase.Status != model.PurchaseStatusPaid {
			if err = markPurchasePaidSimple(txCtx, s.Purchases, purchase); err != nil {
				return err
			}
		}
		locked, err = s.commitPendingPlan(txCtx, locked, newPlanID, false)
		if err != nil {
			return err
		}
		locked.PlanChangeDate = nil
		locked.UpdatedAt = now
		if err = s.Supporters.Update(txCtx, &locked); err != nil {
			return err
		}
		supporter = locked
		return nil
	})
	if err != nil {
		return SupporterView{}, err
	}
	return s.enrichView(ctx, supporter)
}
