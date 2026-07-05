package usecase

import (
	"context"
	"time"

	"crowdfund-app/model"
	"crowdfund-app/repository"
	"crowdfund-app/stripe"

	"github.com/google/uuid"
)

type SupporterUseCase struct {
	Tx         repository.TxBeginner
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
	if supporter.PendingBillingMode == nil {
		return false, nil
	}
	err = s.Tx.WithTx(ctx, func(txCtx context.Context) error {
		var locked model.ProjectSupporter
		locked, err = s.Supporters.GetID(txCtx, supporter.ID)
		if err != nil {
			return mapRepoNotFound(err)
		}
		if locked.PlanChangeDate == nil || now.Before(*locked.PlanChangeDate) {
			return nil
		}
		if locked.PendingBillingMode == nil {
			return nil
		}
		_, _, _, err = s.Projects.GetID(txCtx, locked.ProjectID)
		if err != nil {
			return mapRepoNotFound(err)
		}
		if err = s.applyBillingMode(txCtx, locked, *locked.PendingBillingMode); err != nil {
			return err
		}
		locked.BillingMode = *locked.PendingBillingMode
		locked.PendingBillingMode = nil
		locked.PlanChangeDate = nil
		locked.UpdatedAt = now
		return s.Supporters.Update(txCtx, &locked)
	})
	if err != nil {
		return false, err
	}
	return true, nil
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
		v       SupporterView
		project model.Project
		plan    model.ProjectPlan
		err     error
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
