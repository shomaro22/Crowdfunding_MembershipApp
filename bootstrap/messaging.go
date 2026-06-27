package bootstrap

import (
	"crowdfund-app/usecase"
	"crowdfund-app/wshub"
)

func WireMessageEvents(msg *usecase.MessageUseCase, hub *wshub.ConnectionManager) {
	if msg == nil || hub == nil {
		return
	}
	usecase.AttachMessageEvents(msg, wshub.NewMessagePublisher(hub, msg))
}
