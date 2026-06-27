package wshub

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Hub is the WebSocket connection registry used by HTTP controllers.
type Hub interface {
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
		filter SubscriberFilter,
	)
}
