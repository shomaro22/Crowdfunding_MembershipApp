package controller

import (
	"context"

	"crowdfund-app/usecase"
	"crowdfund-app/wshub"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type WebSocketHub interface {
	Register(userID uuid.UUID, conn *websocket.Conn) uuid.UUID
	RemoveConn(connID uuid.UUID)
	Subscribe(connID, projectID uuid.UUID)
	Unsubscribe(connID, projectID uuid.UUID)
	ProjectIDs(connID uuid.UUID) []uuid.UUID
	WriteText(connID uuid.UUID, payload []byte) bool
	SendPing(connID uuid.UUID) bool
	BroadcastToProjectExceptUserFiltered(
		projectID uuid.UUID,
		payload []byte,
		excludeUserID uuid.UUID,
		filter wshub.SubscriberFilter,
	)
}

type MessageProjectAccess interface {
	CanSubscribeProject(ctx context.Context, projectID, userID uuid.UUID) error
	ConfirmWSSubscribe(ctx context.Context, userID, projectID uuid.UUID, meta usecase.RequestMeta) error
}
