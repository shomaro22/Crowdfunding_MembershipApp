package bootstrap

import (
	"crowdfund-app/wshub"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type WSHubAdapter struct {
	hub *wshub.ConnectionManager
}

func NewWSHubAdapter(hub *wshub.ConnectionManager) WSHubAdapter {
	return WSHubAdapter{hub: hub}
}

func (a WSHubAdapter) Register(userID uuid.UUID, conn *websocket.Conn) uuid.UUID {
	if a.hub == nil {
		return uuid.Nil
	}
	return a.hub.Register(userID, conn)
}

func (a WSHubAdapter) RemoveConn(connID uuid.UUID) {
	if a.hub == nil {
		return
	}
	a.hub.RemoveConn(connID)
}

func (a WSHubAdapter) Subscribe(connID, projectID uuid.UUID) {
	if a.hub == nil {
		return
	}
	a.hub.Subscribe(connID, projectID)
}

func (a WSHubAdapter) Unsubscribe(connID, projectID uuid.UUID) {
	if a.hub == nil {
		return
	}
	a.hub.Unsubscribe(connID, projectID)
}

func (a WSHubAdapter) WriteText(connID uuid.UUID, payload []byte) bool {
	if a.hub == nil {
		return false
	}
	return a.hub.WriteText(connID, payload)
}

func (a WSHubAdapter) SendPing(connID uuid.UUID) bool {
	if a.hub == nil {
		return false
	}
	return a.hub.SendPing(connID)
}

func (a WSHubAdapter) BroadcastToProject(projectID uuid.UUID, payload []byte) {
	if a.hub == nil {
		return
	}
	a.hub.BroadcastToProject(projectID, payload)
}

func (a WSHubAdapter) BroadcastToProjectExceptUser(projectID uuid.UUID, payload []byte, excludeUserID uuid.UUID) {
	if a.hub == nil {
		return
	}
	a.hub.BroadcastToProjectExceptUser(projectID, payload, excludeUserID)
}
