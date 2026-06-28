package usecase

import (
	"context"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"

	"github.com/google/uuid"
)

func (s *SupporterUseCase) EnrichSupporterRows(ctx context.Context, rows []entity.ProjectSupporter) ([]SupporterView, error) {
	return s.enrichViews(ctx, rows)
}

func (s *SupporterUseCase) ListMine(ctx context.Context, userID uuid.UUID) ([]SupporterView, error) {
	var (
		rows []entity.ProjectSupporter
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
		rows []entity.ProjectSupporter
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
		ok, err = s.applyPendingChanges(ctx, &rows[i])
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

func (s *SupporterUseCase) applyPendingChanges(ctx context.Context, supporter *entity.ProjectSupporter) (bool, error) {
	var (
		now time.Time
		due bool
		err error
	)
	if supporter == nil {
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
		var (
			locked *entity.ProjectSupporter
			project *entity.Project
		)
		locked, err = s.Supporters.FindID(txCtx, supporter.ID)
		if err != nil || locked == nil {
			return ErrNotFound
		}
		if locked.PlanChangeDate == nil || now.Before(*locked.PlanChangeDate) {
			*supporter = *locked
			return nil
		}
		project, _, _, err = s.Projects.FindID(txCtx, locked.ProjectID)
		if err != nil || project == nil {
			return ErrNotFound
		}
		if locked.PendingPlanID != nil {
			if err = s.commitPendingPlan(txCtx, locked, *locked.PendingPlanID, true); err != nil {
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
		if err = s.Supporters.Update(txCtx, locked); err != nil {
			return err
		}
		*supporter = *locked
		return nil
	})
	if err != nil {
		return false, err
	}
	return due, nil
}

func (s *SupporterUseCase) enrichViews(ctx context.Context, rows []entity.ProjectSupporter) ([]SupporterView, error) {
	var (
		out []SupporterView
		i   int
		v   SupporterView
		err error
	)
	out = make([]SupporterView, 0, len(rows))
	for i = range rows {
		v, err = s.enrichView(ctx, &rows[i])
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (s *SupporterUseCase) enrichView(ctx context.Context, supporter *entity.ProjectSupporter) (SupporterView, error) {
	var (
		v       SupporterView
		project *entity.Project
		plan    *entity.ProjectPlan
		err     error
	)
	if supporter == nil {
		return SupporterView{}, ErrNotFound
	}
	v.Supporter = SupporterItemFrom(supporter)
	project, _, _, err = s.Projects.FindID(ctx, supporter.ProjectID)
	if err != nil {
		return SupporterView{}, err
	}
	v.Project = ProjectItemPtrFrom(project)
	plan, err = s.Projects.FindPlan(ctx, supporter.ProjectID, supporter.PlanID)
	if err != nil {
		return SupporterView{}, err
	}
	v.Plan = ProjectPlanPtrFrom(plan)
	if supporter.PendingPlanID != nil {
		plan, err = s.Projects.FindPlan(ctx, supporter.ProjectID, *supporter.PendingPlanID)
		if err != nil {
			return SupporterView{}, err
		}
		v.PendingPlan = ProjectPlanPtrFrom(plan)
	}
	return v, nil
}

func (s *SupporterUseCase) loadOwned(ctx context.Context, userID, supporterID uuid.UUID) (*entity.ProjectSupporter, error) {
	var (
		row *entity.ProjectSupporter
		err error
	)
	row, err = s.Supporters.FindOwned(ctx, supporterID, userID)
	if err != nil || row == nil {
		return nil, ErrNotFound
	}
	return row, nil
}

func (s *SupporterUseCase) commitPendingPlan(
	ctx context.Context,
	locked *entity.ProjectSupporter,
	newPlanID uuid.UUID,
	validatePlan bool,
) error {
	var (
		oldPlanID uuid.UUID
		newPlan   *entity.ProjectPlan
		plan      *entity.ProjectPlan
		err       error
	)
	if validatePlan {
		newPlan, err = s.Projects.FindPlan(ctx, locked.ProjectID, newPlanID)
		if err != nil || newPlan == nil || !newPlan.IsActive {
			return ErrBadRequest
		}
	}
	if err = s.Projects.TransferPlanSupportCount(ctx, locked.ProjectID, locked.PlanID, newPlanID); err != nil {
		return err
	}
	oldPlanID = locked.PlanID
	if locked.BillingMode == entity.BillingModeAuto && locked.StripeSubscriptionID != nil && s.Stripe != nil {
		plan, err = s.Projects.FindPlan(ctx, locked.ProjectID, newPlanID)
		if err != nil || plan == nil {
			return ErrNotFound
		}
		if err = s.Stripe.UpdateSubscriptionAmount(ctx, *locked.StripeSubscriptionID, int64(plan.Price)); err != nil {
			return err
		}
	}
	locked.PlanID = newPlanID
	locked.PendingPlanID = nil
	if newPlanID != oldPlanID {
		LogActivity(ctx, s.Activity, &locked.UserID, entity.PurchasePlanChange, "supporter", locked.ID.String(), RequestMeta{}, activityMeta(map[string]string{
			"from_plan_id": oldPlanID.String(),
			"to_plan_id":   newPlanID.String(),
		}))
	}
	return nil
}
