package events

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SupportCompleted は初回支援（購入完了）を表すドメインイベント。
type SupportCompleted struct {
	PurchaseID  uuid.UUID
	ProjectID   uuid.UUID
	BuyerID     uuid.UUID
	SupporterID uuid.UUID
	PlanID      uuid.UUID
	Amount      int64
	PeriodStart time.Time
	PeriodEnd   time.Time
}

// Handler は同期イベントハンドラ。
type Handler func(ctx context.Context, e SupportCompleted) error

// Bus は SupportCompleted の同期イベントバス。
type Bus struct {
	mu       sync.RWMutex
	handlers []Handler
}

// NewBus は空のイベントバスを返す。
func NewBus() *Bus {
	return &Bus{}
}

// Subscribe はハンドラを登録する。
func (b *Bus) Subscribe(h Handler) {
	b.mu.Lock()
	b.handlers = append(b.handlers, h)
	b.mu.Unlock()
}

// Publish は登録済みハンドラを順に実行する。最初のエラーで停止する。
func (b *Bus) Publish(ctx context.Context, e SupportCompleted) error {
	var (
		handlers []Handler
		i        int
		err      error
	)

	b.mu.RLock()
	handlers = make([]Handler, len(b.handlers))
	copy(handlers, b.handlers)
	b.mu.RUnlock()

	for i = range handlers {
		if err = handlers[i](ctx, e); err != nil {
			return err
		}
	}
	return nil
}
