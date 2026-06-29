package usecase

import (
	"context"
	"time"

	"crowdfund-app/model"
	"crowdfund-app/repository"

	"github.com/google/uuid"
)

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
