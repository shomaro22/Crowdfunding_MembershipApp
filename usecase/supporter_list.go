package usecase

import (
	"context"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"

	"github.com/google/uuid"
)

func (s *SupporterUseCase) ListMine(ctx context.Context, userID uuid.UUID) ([]SupporterView, error) {
	var (
		rows []entity.ProjectSupporter
		err  error
		i    int
	)
	rows, err = s.Supporters.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i = range rows {
		var ok bool
		ok, err = s.applyPendingChanges(ctx, &rows[i])
		if err != nil {
			return nil, err
		}
		_ = ok
	}
	return s.enrichViews(ctx, rows)
}

func (s *SupporterUseCase) ApplyPlans(ctx context.Context) (int, error) {
	var (
		rows    []entity.ProjectSupporter
		applied int
		err     error
		i       int
		ok      bool
	)
	rows, err = s.Supporters.ListDuePlanChanges(ctx, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	for i = range rows {
		ok, err = s.applyPendingChanges(ctx, &rows[i])
		if err != nil {
			return applied, err
		}
		if ok {
			applied++
		}
	}
	return applied, nil
}

func (s *SupporterUseCase) applyPendingChanges(ctx context.Context, supporter *entity.ProjectSupporter) (bool, error) {
	var (
		now   time.Time
		due   bool
		err   error
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
			locked  *entity.ProjectSupporter
			proj    *entity.Project
			newPlan *entity.ProjectPlan
		)
		locked, err = s.Supporters.FindID(txCtx, supporter.ID)
		if err != nil || locked == nil {
			return ErrNotFound
		}
		if locked.PlanChangeDate == nil || now.Before(*locked.PlanChangeDate) {
			*supporter = *locked
			return nil
		}
		proj, _, _, err = s.Projects.FindID(txCtx, locked.ProjectID)
		if err != nil || proj == nil {
			return ErrNotFound
		}
		if locked.PendingPlanID != nil {
			var oldPlanID uuid.UUID
			oldPlanID = locked.PlanID
			newPlan, err = s.Projects.FindPlan(txCtx, locked.ProjectID, *locked.PendingPlanID)
			if err != nil || newPlan == nil || !newPlan.IsActive {
				return ErrBadRequest
			}
			if err = s.Projects.TransferPlanSupportCount(txCtx, locked.ProjectID, locked.PlanID, newPlan.ID); err != nil {
				return err
			}
			if locked.BillingMode == entity.BillingModeAuto && locked.StripeSubscriptionID != nil && s.Stripe != nil {
				if err = s.Stripe.UpdateSubscriptionAmount(txCtx, *locked.StripeSubscriptionID, int64(newPlan.Price)); err != nil {
					return err
				}
			}
			locked.PlanID = newPlan.ID
			locked.PendingPlanID = nil
			if newPlan.ID != oldPlanID {
				LogActivity(txCtx, s.Activity, &locked.UserID, entity.PurchasePlanChange, "supporter", locked.ID.String(), RequestMeta{}, activityMeta(map[string]string{
					"from_plan_id": oldPlanID.String(),
					"to_plan_id":   newPlan.ID.String(),
				}))
			}
		}
		if locked.PendingBillingMode != nil {
			if err = s.applyBillingModeSwitch(txCtx, locked, *locked.PendingBillingMode); err != nil {
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
		v   SupporterView
		proj *entity.Project
		err error
	)
	if supporter == nil {
		return SupporterView{}, ErrNotFound
	}
	v.Supporter = supporter
	proj, _, _, err = s.Projects.FindID(ctx, supporter.ProjectID)
	if err != nil {
		return SupporterView{}, err
	}
	v.Project = proj
	v.Plan, err = s.Projects.FindPlan(ctx, supporter.ProjectID, supporter.PlanID)
	if err != nil {
		return SupporterView{}, err
	}
	if supporter.PendingPlanID != nil {
		v.PendingPlan, err = s.Projects.FindPlan(ctx, supporter.ProjectID, *supporter.PendingPlanID)
		if err != nil {
			return SupporterView{}, err
		}
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

func supporterCanManage(s *entity.ProjectSupporter, now time.Time) bool {
	if s == nil {
		return false
	}
	switch s.Status {
	case entity.SupporterStatusActive, entity.SupporterStatusPendingRenewal, entity.SupporterStatusCancelled:
		return s.HasBenefits(now)
	default:
		return false
	}
}
