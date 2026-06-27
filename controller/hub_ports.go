package controller

import (
	"context"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type ProjectBroadcastHub interface {
	BroadcastToProject(projectID uuid.UUID, payload []byte)
	BroadcastToProjectExceptUser(projectID uuid.UUID, payload []byte, excludeUserID uuid.UUID)
}

type WebSocketHub interface {
	Register(userID uuid.UUID, conn *websocket.Conn) uuid.UUID
	RemoveConn(connID uuid.UUID)
	Subscribe(connID, projectID uuid.UUID)
	Unsubscribe(connID, projectID uuid.UUID)
	WriteText(connID uuid.UUID, payload []byte) bool
	SendPing(connID uuid.UUID) bool
	BroadcastToProject(projectID uuid.UUID, payload []byte)
	BroadcastToProjectExceptUser(projectID uuid.UUID, payload []byte, excludeUserID uuid.UUID)
}

type MessageProjectAccess interface {
	CanSubscribeProject(ctx context.Context, projectID, userID uuid.UUID) error
}
