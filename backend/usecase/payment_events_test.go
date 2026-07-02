package usecase_test

import (
	"context"
	"testing"
	"time"

	"crowdfund-app/events"
	"crowdfund-app/model"
	"crowdfund-app/repository"
	"crowdfund-app/usecase"

	"github.com/google/uuid"
)

type supportActivityRepo struct {
	lastAction model.ActivityAction
}

func (r *supportActivityRepo) Create(_ context.Context, row *model.ActivityLog) error {
	r.lastAction = row.Action
	return nil
}
func (r *supportActivityRepo) List(_ context.Context, _ repository.ActivityListFilter) ([]model.ActivityLog, int64, error) {
	return nil, 0, nil
}
func (r *supportActivityRepo) CountByActionBetween(_ context.Context, _ model.ActivityAction, _, _ time.Time) (int64, error) {
	return 0, nil
}
func (r *supportActivityRepo) TopIPsBetween(_ context.Context, _, _ time.Time, _ int) ([]repository.ActivityIPCount, error) {
	return nil, nil
}

func TestSupportActivityHandlerLogsPurchaseCreated(t *testing.T) {
	var (
		activity supportActivityRepo
		bus      = events.NewBus()
		err      error
	)

	bus.Subscribe(usecase.SupportActivityHandler(&activity))
	err = bus.Publish(context.Background(), events.SupportCompleted{
		PurchaseID:  uuid.New(),
		ProjectID:   uuid.New(),
		BuyerID:     uuid.New(),
		SupporterID: uuid.New(),
		PlanID:      uuid.New(),
		Amount:      3000,
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if activity.lastAction != model.PurchaseCreated {
		t.Fatalf("action=%q want purchase.created", activity.lastAction)
	}
}
