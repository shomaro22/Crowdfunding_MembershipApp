package bootstrap

import (
	"crowdfund-app/events"
	"crowdfund-app/repository"
	"crowdfund-app/usecase"
)

// NewSupportEventBus は支援完了イベントバスを構築する。
func NewSupportEventBus(activity repository.IActivityRepo) *events.Bus {
	var bus = events.NewBus()
	bus.Subscribe(usecase.SupportActivityHandler(activity))
	return bus
}

// AttachSupportEvents は決済ユースケースにイベントバスを接続する。
func AttachSupportEvents(payment usecase.IPaymentUsecase, bus *events.Bus) {
	var raw *usecase.PaymentUseCase
	if bus == nil {
		return
	}
	raw, ok := payment.(*usecase.PaymentUseCase)
	if !ok {
		return
	}
	usecase.AttachSupportEvents(raw, bus)
}
