package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

type WSController struct {
	Hub      WebSocketHub
	Messages MessageProjectAccess
}

func NewWSController(hub WebSocketHub, messages MessageProjectAccess) *WSController {
	return &WSController{Hub: hub, Messages: messages}
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

const wsDBOpTimeout = 5 * time.Second

func (w *WSController) Handle(c echo.Context) error {
	var (
		userID uuid.UUID
		conn   *websocket.Conn
		err    error
	)
	if w.Hub == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, apiError("websocket hub not configured", "INTERNAL_ERROR"))
	}
	userID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	conn, err = wsUpgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(WSReadDeadline))
	})
	conn.SetReadDeadline(time.Now().Add(WSReadDeadline))

	connID := w.Hub.Register(userID, conn)
	w.writeJSON(connID, wsSessionReadyOutbound{
		Type:          WSEventSessionReady,
		SchemaVersion: WSEventSchemaVersion,
		UserID:        userID,
	})

	done := make(chan struct{})
	go w.pingLoop(connID, done)

	defer func() {
		close(done)
		w.Hub.RemoveConn(connID)
	}()

	for {
		var (
			data []byte
			in   wsClientMessage
		)
		_, data, err = conn.ReadMessage()
		if err != nil {
			break
		}
		_ = conn.SetReadDeadline(time.Now().Add(WSReadDeadline))
		if err = json.Unmarshal(data, &in); err != nil {
			w.writeWSError(connID, WSCodeInvalidJSON, "invalid json", nil)
			continue
		}
		if in.SchemaVersion != 0 && in.SchemaVersion != WSClientSchemaVersion {
			w.writeWSError(connID, WSCodeSchemaVersion, "unsupported schema_version", nil)
			continue
		}
		w.dispatchClientFrame(c, connID, userID, in)
	}
	return nil
}

func (w *WSController) dispatchClientFrame(c echo.Context, connID, userID uuid.UUID, in wsClientMessage) {
	var (
		typ        string
		projectRaw string
		projectID  uuid.UUID
		err        error
	)
	typ = strings.TrimSpace(in.Type)
	if typ == "" {
		w.writeWSError(connID, WSCodeMissingType, "type is required", nil)
		return
	}
	if typ == WSClientPing {
		w.writeJSON(connID, wsPongOutbound{Type: WSEventPong, SchemaVersion: WSEventSchemaVersion})
		return
	}
	projectRaw = strings.TrimSpace(in.ProjectID)
	if projectRaw == "" {
		w.writeWSError(connID, WSCodeMissingProjectID, "project_id is required", nil)
		return
	}
	projectID, err = uuid.Parse(projectRaw)
	if err != nil {
		w.writeWSError(connID, WSCodeInvalidProjectID, "invalid project_id", nil)
		return
	}
	projectPtr := projectID.String()
	switch typ {
	case WSClientTyping:
		w.handleTyping(connID, userID, projectID, projectPtr, in.IsTyping)
	case WSClientSubscribe:
		w.handleSubscribe(c, connID, userID, projectID, projectPtr)
	case WSClientUnsubscribe:
		w.Hub.Unsubscribe(connID, projectID)
		w.writeJSON(connID, wsProjectAckOutbound{
			Type:          WSEventUnsubscribed,
			SchemaVersion: WSEventSchemaVersion,
			ProjectID:     projectPtr,
		})
	default:
		w.writeWSError(connID, WSCodeUnknownType, "unknown type", &projectPtr)
	}
}

func (w *WSController) handleSubscribe(c echo.Context, connID, userID, projectID uuid.UUID, projectPtr string) {
	var (
		ctx    context.Context
		cancel context.CancelFunc
		err    error
	)
	ctx, cancel = context.WithTimeout(c.Request().Context(), wsDBOpTimeout)
	err = w.Messages.CanSubscribeProject(ctx, projectID, userID)
	cancel()
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrForbidden):
			w.writeWSError(connID, WSCodeForbiddenNotMember, "not allowed to subscribe to this project", &projectPtr)
		case errors.Is(err, usecase.ErrNotFound):
			w.writeWSError(connID, WSCodeInvalidProjectID, "project not found", &projectPtr)
		default:
			w.writeWSError(connID, WSCodeInternal, "internal error", &projectPtr)
		}
		return
	}
	w.Hub.Subscribe(connID, projectID)
	w.writeJSON(connID, wsProjectAckOutbound{
		Type:          WSEventSubscribed,
		SchemaVersion: WSEventSchemaVersion,
		ProjectID:     projectPtr,
	})
}

func (w *WSController) handleTyping(connID, userID, projectID uuid.UUID, projectPtr string, isTyping *bool) {
	var (
		ctx    context.Context
		cancel context.CancelFunc
		active bool
		payload []byte
		err    error
	)
	active = true
	if isTyping != nil {
		active = *isTyping
	}
	ctx, cancel = context.WithTimeout(context.Background(), wsDBOpTimeout)
	err = w.Messages.CanSubscribeProject(ctx, projectID, userID)
	cancel()
	if err != nil {
		w.writeWSError(connID, WSCodeForbiddenNotMember, "not allowed", &projectPtr)
		return
	}
	payload, err = json.Marshal(typingOutboundPayload{
		Type:          WSEventTyping,
		SchemaVersion: WSEventSchemaVersion,
		ProjectID:     projectID,
		UserID:        userID,
		IsTyping:      active,
	})
	if err == nil {
		w.Hub.BroadcastToProjectExceptUser(projectID, payload, userID)
	}
}

func (w *WSController) pingLoop(connID uuid.UUID, done <-chan struct{}) {
	var ticker *time.Ticker
	ticker = time.NewTicker(WSPingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if !w.Hub.SendPing(connID) {
				return
			}
		}
	}
}

func (w *WSController) writeWSError(connID uuid.UUID, code, msg string, projectID *string) {
	w.writeJSON(connID, wsErrorOutbound{
		Type:          WSEventError,
		SchemaVersion: WSEventSchemaVersion,
		Code:          code,
		Message:       msg,
		ProjectID:     projectID,
	})
}

func (w *WSController) writeJSON(connID uuid.UUID, v any) {
	var (
		b   []byte
		err error
	)
	b, err = json.Marshal(v)
	if err != nil {
		return
	}
	_ = w.Hub.WriteText(connID, b)
}
