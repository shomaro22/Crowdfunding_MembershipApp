package usecase

import (
	"context"
	"strconv"

	"crowdfund-app/events"
	"crowdfund-app/model"
	"crowdfund-app/repository"
)

// AttachSupportEvents は決済ユースケースに支援完了イベントバスを接続する。
func AttachSupportEvents(p *PaymentUseCase, bus *events.Bus) {
	if p == nil || bus == nil {
		return
	}
	p.supportEvents = bus
}

// SupportActivityHandler は支援完了時に activity_log を記録する。
func SupportActivityHandler(activity repository.IActivityRepo) events.Handler {
	return func(ctx context.Context, e events.SupportCompleted) error {
		LogActivity(ctx, activity, &e.BuyerID, model.PurchaseCreated, "purchase", e.PurchaseID.String(), RequestMeta{},
			activityMeta(map[string]string{
				"project_id":   e.ProjectID.String(),
				"supporter_id": e.SupporterID.String(),
				"plan_id":      e.PlanID.String(),
				"amount":       strconv.FormatInt(e.Amount, 10),
			}))
		return nil
	}
}
