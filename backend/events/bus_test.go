package events

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBusPublishInvokesHandlersInOrder(t *testing.T) {
	var (
		bus    = NewBus()
		order  []int
		e      SupportCompleted
		err    error
	)

	bus.Subscribe(func(_ context.Context, _ SupportCompleted) error {
		order = append(order, 1)
		return nil
	})
	bus.Subscribe(func(_ context.Context, _ SupportCompleted) error {
		order = append(order, 2)
		return nil
	})

	e = SupportCompleted{
		PurchaseID:  uuid.New(),
		ProjectID:   uuid.New(),
		BuyerID:     uuid.New(),
		SupporterID: uuid.New(),
		PlanID:      uuid.New(),
		Amount:      1000,
		PeriodStart: time.Now().UTC(),
		PeriodEnd:   time.Now().UTC().AddDate(0, 0, 30),
	}
	err = bus.Publish(context.Background(), e)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Fatalf("handler order=%v want [1 2]", order)
	}
}

func TestBusPublishStopsOnFirstError(t *testing.T) {
	var (
		bus   = NewBus()
		calls int
		want  = errors.New("handler failed")
	)

	bus.Subscribe(func(_ context.Context, _ SupportCompleted) error {
		calls++
		return want
	})
	bus.Subscribe(func(_ context.Context, _ SupportCompleted) error {
		calls++
		return nil
	})

	err := bus.Publish(context.Background(), SupportCompleted{})
	if !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
	if calls != 1 {
		t.Fatalf("calls=%d want 1", calls)
	}
}

func TestBusPublishNoHandlers(t *testing.T) {
	var bus = NewBus()
	if err := bus.Publish(context.Background(), SupportCompleted{}); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}
