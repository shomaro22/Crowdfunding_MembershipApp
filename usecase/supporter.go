package usecase

import (
	"context"
	"time"

	"crowdfund-app/model"
	"crowdfund-app/repository"
	"crowdfund-app/stripe"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UpgradeOut struct {
	Supporter                 SupporterView
	PaymentIntentClientSecret *string
}

type SupporterUseCase struct {
	DB         *gorm.DB
	Users      repository.IUserRepo
	Projects   repository.IProjectRepo
	Supporters repository.ISupporterRepo
	Purchases  repository.IPurchaseRepo
	Connect    repository.IConnectRepo
	Activity   repository.IActivityRepo
	Stripe     stripe.Client
}

type ChangeBillingIn struct {
	BillingMode   string
	PaymentMethod string
}

func (s *SupporterUseCase) Cancel(ctx context.Context, userID, supporterID uuid.UUID) (SupporterView, error) {
	var (
		supporter model.ProjectSupporter
		now       time.Time
		err       error
	)
	now = time.Now().UTC()
	supporter, err = s.loadOwned(ctx, userID, supporterID)
	if err != nil {
		return SupporterView{}, err
	}
	if !supporterCanManage(supporter, now) || supporter.Status == model.SupporterStatusCancelled {
		return SupporterView{}, ErrForbidden
	}
	if supporter.BillingMode == model.BillingModeAuto &&
		supporter.StripeSubscriptionID != nil &&
		s.Stripe != nil {
		if err = s.Stripe.CancelSubscriptionAtPeriodEnd(ctx, *supporter.StripeSubscriptionID); err != nil {
			return SupporterView{}, err
		}
	}
	supporter.Status = model.SupporterStatusCancelled
	supporter.UpdatedAt = now
	if err = s.Supporters.Update(ctx, &supporter); err != nil {
		return SupporterView{}, err
	}
	return s.enrichView(ctx, supporter)
}

func (s *SupporterUseCase) Rejoin(ctx context.Context, userID, supporterID uuid.UUID) (SupporterView, error) {
	var (
		supporter model.ProjectSupporter
		now       time.Time
		err       error
	)
	now = time.Now().UTC()
	supporter, err = s.loadOwned(ctx, userID, supporterID)
	if err != nil {
		return SupporterView{}, err
	}
	if supporter.Status != model.SupporterStatusCancelled {
		return SupporterView{}, ErrBadRequest
	}
	if now.After(supporter.CurrentPeriodEnd) {
		return SupporterView{}, ErrBadRequest
	}
	if supporter.BillingMode == model.BillingModeAuto &&
		supporter.StripeSubscriptionID != nil &&
		s.Stripe != nil {
		if err = s.Stripe.ResumeSubscription(ctx, *supporter.StripeSubscriptionID); err != nil {
			return SupporterView{}, err
		}
	}
	supporter.Status = model.SupporterStatusActive
	supporter.UpdatedAt = now
	if err = s.Supporters.Update(ctx, &supporter); err != nil {
		return SupporterView{}, err
	}
	return s.enrichView(ctx, supporter)
}

func (s *SupporterUseCase) ChangeBilling(ctx context.Context, userID, supporterID uuid.UUID, in ChangeBillingIn) (SupporterView, error) {
	var (
		supporter model.ProjectSupporter
		mode      model.BillingMode
		now       time.Time
		err       error
	)
	mode = model.BillingMode(in.BillingMode)
	if !mode.Valid() {
		return SupporterView{}, ErrBadRequest
	}
	now = time.Now().UTC()
	supporter, err = s.loadOwned(ctx, userID, supporterID)
	if err != nil {
		return SupporterView{}, err
	}
	if !supporterCanManage(supporter, now) || supporter.Status == model.SupporterStatusCancelled {
		return SupporterView{}, ErrForbidden
	}
	if supporter.PendingBillingMode != nil {
		return SupporterView{}, ErrBadRequest
	}
	if mode == supporter.BillingMode {
		return SupporterView{}, ErrBadRequest
	}
	changeAt := supporter.CurrentPeriodEnd
	supporter.PendingBillingMode = &mode
	supporter.PlanChangeDate = &changeAt
	supporter.UpdatedAt = now
	if mode == model.BillingModeKonbini &&
		supporter.BillingMode == model.BillingModeAuto &&
		supporter.StripeSubscriptionID != nil &&
		s.Stripe != nil {
		if err = s.Stripe.CancelSubscriptionAtPeriodEnd(ctx, *supporter.StripeSubscriptionID); err != nil {
			return SupporterView{}, err
		}
	}
	if err = s.Supporters.Update(ctx, &supporter); err != nil {
		return SupporterView{}, err
	}
	return s.enrichView(ctx, supporter)
}

func (s *SupporterUseCase) applyBillingMode(ctx context.Context, supporter model.ProjectSupporter, mode model.BillingMode) error {
	switch mode {
	case model.BillingModeKonbini:
		supporter.StripeSubscriptionID = nil
		return nil
	case model.BillingModeAuto:
		if supporter.StripeSubscriptionID == nil {
			return ErrBadRequest
		}
		if s.Stripe == nil {
			return ErrStripeUnavailable
		}
		return s.Stripe.ResumeSubscription(ctx, *supporter.StripeSubscriptionID)
	default:
		return ErrBadRequest
	}
}

func supporterCanManage(s model.ProjectSupporter, now time.Time) bool {
	switch s.Status {
	case model.SupporterStatusActive, model.SupporterStatusPendingRenewal, model.SupporterStatusCancelled:
		return s.HasBenefits(now)
	default:
		return false
	}
}

func (s *SupporterUseCase) ViewForSupporter(ctx context.Context, supporter model.ProjectSupporter) (SupporterView, error) {
	return s.enrichView(ctx, supporter)
}

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

func (s *SupporterUseCase) EnrichSupporterRows(ctx context.Context, rows []model.ProjectSupporter) ([]SupporterView, error) {
	return s.enrichViews(ctx, rows)
}

func (s *SupporterUseCase) ListMine(ctx context.Context, userID uuid.UUID) ([]SupporterView, error) {
	var (
		rows []model.ProjectSupporter
		err  error
	)
	rows, err = s.Supporters.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.enrichViews(ctx, rows)
}

func (s *SupporterUseCase) RunApplyPlans(ctx context.Context, dryRun bool) (processed, failed int, err error) {
	var (
		rows []model.ProjectSupporter
		now  time.Time
		i    int
		ok   bool
	)
	now = time.Now().UTC()
	rows, err = s.Supporters.ListDuePlanChanges(ctx, now)
	if err != nil {
		return 0, 0, err
	}
	if dryRun {
		return len(rows), 0, nil
	}
	for i = range rows {
		ok, err = s.applyPendingChanges(ctx, rows[i])
		if err != nil {
			failed++
			err = nil
			continue
		}
		if ok {
			processed++
		}
	}
	return processed, failed, nil
}

func (s *SupporterUseCase) applyPendingChanges(ctx context.Context, supporter model.ProjectSupporter) (bool, error) {
	var (
		now time.Time
		due bool
		err error
	)
	if supporter.ID == uuid.Nil {
		return false, nil
	}
	now = time.Now().UTC()
	if supporter.PlanChangeDate == nil {
		return false, nil
	}
	if now.Before(*supporter.PlanChangeDate) {
		return false, nil
	}
	due = supporter.PendingPlanID != nil || supporter.PendingBillingMode != nil
	if !due {
		return false, nil
	}
	err = repository.WithTx(ctx, s.DB, func(txCtx context.Context) error {
		var locked model.ProjectSupporter
		locked, err = s.Supporters.GetID(txCtx, supporter.ID)
		if err != nil {
			return mapRepoNotFound(err)
		}
		if locked.PlanChangeDate == nil || now.Before(*locked.PlanChangeDate) {
			return nil
		}
		_, _, _, err = s.Projects.GetID(txCtx, locked.ProjectID)
		if err != nil {
			return mapRepoNotFound(err)
		}
		if locked.PendingPlanID != nil {
			locked, err = s.commitPendingPlan(txCtx, locked, *locked.PendingPlanID, true)
			if err != nil {
				return err
			}
		}
		if locked.PendingBillingMode != nil {
			if err = s.applyBillingMode(txCtx, locked, *locked.PendingBillingMode); err != nil {
				return err
			}
			locked.BillingMode = *locked.PendingBillingMode
			locked.PendingBillingMode = nil
		}
		locked.PlanChangeDate = nil
		locked.UpdatedAt = now
		return s.Supporters.Update(txCtx, &locked)
	})
	if err != nil {
		return false, err
	}
	return due, nil
}

func (s *SupporterUseCase) enrichViews(ctx context.Context, rows []model.ProjectSupporter) ([]SupporterView, error) {
	var (
		out []SupporterView
		i   int
		v   SupporterView
		err error
	)
	out = make([]SupporterView, 0, len(rows))
	for i = range rows {
		v, err = s.enrichView(ctx, rows[i])
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (s *SupporterUseCase) enrichView(ctx context.Context, supporter model.ProjectSupporter) (SupporterView, error) {
	var (
		v           SupporterView
		project     model.Project
		plan        model.ProjectPlan
		pendingPlan model.ProjectPlan
		err         error
	)
	if supporter.ID == uuid.Nil {
		return SupporterView{}, ErrNotFound
	}
	v.Supporter = SupporterItemFrom(supporter)
	project, _, _, err = s.Projects.GetID(ctx, supporter.ProjectID)
	if err != nil {
		return SupporterView{}, mapRepoNotFound(err)
	}
	v.Project = ProjectItemPtrFrom(project)
	plan, err = s.Projects.GetPlan(ctx, supporter.ProjectID, supporter.PlanID)
	if err != nil {
		return SupporterView{}, mapRepoNotFound(err)
	}
	v.Plan = ProjectPlanPtrFrom(plan)
	if supporter.PendingPlanID != nil {
		pendingPlan, err = s.Projects.GetPlan(ctx, supporter.ProjectID, *supporter.PendingPlanID)
		if err != nil {
			return SupporterView{}, mapRepoNotFound(err)
		}
		v.PendingPlan = ProjectPlanPtrFrom(pendingPlan)
	}
	return v, nil
}

func (s *SupporterUseCase) loadOwned(ctx context.Context, userID, supporterID uuid.UUID) (model.ProjectSupporter, error) {
	var (
		row model.ProjectSupporter
		err error
	)
	row, err = s.Supporters.GetOwned(ctx, supporterID, userID)
	if err != nil {
		return model.ProjectSupporter{}, mapRepoNotFound(err)
	}
	return row, nil
}

func (s *SupporterUseCase) commitPendingPlan(
	ctx context.Context,
	locked model.ProjectSupporter,
	newPlanID uuid.UUID,
	validatePlan bool,
) (model.ProjectSupporter, error) {
	var (
		oldPlanID uuid.UUID
		newPlan   model.ProjectPlan
		plan      model.ProjectPlan
		err       error
	)
	if validatePlan {
		newPlan, err = s.Projects.GetPlan(ctx, locked.ProjectID, newPlanID)
		if err != nil {
			return model.ProjectSupporter{}, mapRepoNotFound(err)
		}
		if !newPlan.IsActive {
			return model.ProjectSupporter{}, ErrBadRequest
		}
	}
	if err = s.Projects.TransferPlanSupportCount(ctx, locked.ProjectID, locked.PlanID, newPlanID); err != nil {
		return model.ProjectSupporter{}, err
	}
	oldPlanID = locked.PlanID
	if locked.BillingMode == model.BillingModeAuto && locked.StripeSubscriptionID != nil && s.Stripe != nil {
		plan, err = s.Projects.GetPlan(ctx, locked.ProjectID, newPlanID)
		if err != nil {
			return model.ProjectSupporter{}, mapRepoNotFound(err)
		}
		if err = s.Stripe.UpdateSubscriptionAmount(ctx, *locked.StripeSubscriptionID, int64(plan.Price)); err != nil {
			return model.ProjectSupporter{}, err
		}
	}
	locked.PlanID = newPlanID
	locked.PendingPlanID = nil
	if newPlanID != oldPlanID {
		LogActivity(ctx, s.Activity, &locked.UserID, model.PurchasePlanChange, "supporter", locked.ID.String(), RequestMeta{}, activityMeta(map[string]string{
			"from_plan_id": oldPlanID.String(),
			"to_plan_id":   newPlanID.String(),
		}))
	}
	return locked, nil
}

func isExpirableStatus(status model.SupporterStatus) bool {
	switch status {
	case model.SupporterStatusActive, model.SupporterStatusPendingRenewal, model.SupporterStatusCancelled:
		return true
	default:
		return false
	}
}

func (s *SupporterUseCase) RunExpireSupporters(ctx context.Context, dryRun bool) (processed, failed int, err error) {
	var (
		rows []model.ProjectSupporter
		now  time.Time
		i    int
	)
	now = time.Now().UTC()
	rows, err = s.Supporters.ListExpirable(ctx, now)
	if err != nil {
		return 0, 0, err
	}
	if dryRun {
		return len(rows), 0, nil
	}
	for i = range rows {
		if err = s.expireSupporter(ctx, rows[i], now); err != nil {
			failed++
			continue
		}
		processed++
	}
	return processed, failed, nil
}

func (s *SupporterUseCase) expireSupporter(ctx context.Context, supporter model.ProjectSupporter, now time.Time) error {
	var (
		row model.ProjectSupporter
		err error
	)
	if supporter.ID == uuid.Nil {
		return ErrNotFound
	}
	row, err = s.Supporters.GetID(ctx, supporter.ID)
	if err != nil {
		return mapRepoNotFound(err)
	}
	if row.Status == model.SupporterStatusExpired {
		return nil
	}
	if !now.After(row.CurrentPeriodEnd) || !isExpirableStatus(row.Status) {
		return nil
	}
	row.Status = model.SupporterStatusExpired
	row.UpdatedAt = now
	return s.Supporters.Update(ctx, &row)
}
