package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"crowdfund-app/apierr"
	"crowdfund-app/wshub"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

type WSController struct {
	Hub            wshub.Hub
	Messages       MessageProjectAccess
	allowedOrigins map[string]struct{}
}

func NewWSController(hub wshub.Hub, messages MessageProjectAccess, allowedOrigins []string) *WSController {
	var allowed map[string]struct{}
	if len(allowedOrigins) > 0 {
		allowed = make(map[string]struct{}, len(allowedOrigins))
		for _, origin := range allowedOrigins {
			if origin = strings.TrimSpace(origin); origin != "" {
				allowed[origin] = struct{}{}
			}
		}
	}
	return &WSController{Hub: hub, Messages: messages, allowedOrigins: allowed}
}

const wsDBOpTimeout = 5 * time.Second

func (w *WSController) Handle(c echo.Context) error {
	var (
		userID   uuid.UUID
		conn     *websocket.Conn
		upgrader websocket.Upgrader
		err      error
	)
	if w.Hub == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, apiError("websocket hub not configured", "INTERNAL_ERROR"))
	}
	userID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     w.checkOrigin,
	}
	conn, err = upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	conn.SetReadLimit(wshub.MaxReadBytes)
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(wshub.ReadDeadline))
	})
	conn.SetReadDeadline(time.Now().Add(wshub.ReadDeadline))

	connID := w.Hub.Register(userID, conn)
	writeWSFrame(w.Hub, connID, wshub.SessionReadyOutbound{
		Type:          wshub.EventSessionReady,
		SchemaVersion: wshub.EventSchemaVersion,
		UserID:        userID,
	})

	done := make(chan struct{})
	go w.pingLoop(connID, userID, done)

	defer func() {
		close(done)
		w.Hub.RemoveConn(connID)
	}()

	for {
		var (
			data []byte
			in   wshub.ClientMessage
		)
		_, data, err = conn.ReadMessage()
		if err != nil {
			break
		}
		_ = conn.SetReadDeadline(time.Now().Add(wshub.ReadDeadline))
		if err = json.Unmarshal(data, &in); err != nil {
			w.writeWSError(connID, wshub.CodeInvalidJSON, "invalid json", nil)
			continue
		}
		if in.SchemaVersion != 0 && in.SchemaVersion != wshub.ClientSchemaVersion {
			w.writeWSError(connID, wshub.CodeSchemaVersion, "unsupported schema_version", nil)
			continue
		}
		w.dispatchClientFrame(c, connID, userID, in)
	}
	return nil
}

func (w *WSController) checkOrigin(r *http.Request) bool {
	var origin string
	origin = strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	if len(w.allowedOrigins) == 0 {
		return false
	}
	_, ok := w.allowedOrigins[origin]
	return ok
}

func (w *WSController) dispatchClientFrame(c echo.Context, connID, userID uuid.UUID, in wshub.ClientMessage) {
	var (
		typ        string
		projectRaw string
		projectID  uuid.UUID
		err        error
	)
	typ = strings.TrimSpace(in.Type)
	if typ == "" {
		w.writeWSError(connID, wshub.CodeMissingType, "type is required", nil)
		return
	}
	if typ == wshub.ClientPing {
		writeWSFrame(w.Hub, connID, wshub.PongOutbound{Type: wshub.EventPong, SchemaVersion: wshub.EventSchemaVersion})
		return
	}
	projectRaw = strings.TrimSpace(in.ProjectID)
	if projectRaw == "" {
		w.writeWSError(connID, wshub.CodeMissingProjectID, "project_id is required", nil)
		return
	}
	projectID, err = uuid.Parse(projectRaw)
	if err != nil {
		w.writeWSError(connID, wshub.CodeInvalidProjectID, "invalid project_id", nil)
		return
	}
	projectPtr := projectID.String()
	switch typ {
	case wshub.ClientTyping:
		w.handleTyping(connID, userID, projectID, projectPtr, in.IsTyping)
	case wshub.ClientSubscribe:
		w.handleSubscribe(c, connID, userID, projectID, projectPtr)
	case wshub.ClientUnsubscribe:
		w.Hub.Unsubscribe(connID, projectID)
		writeWSFrame(w.Hub, connID, wshub.ProjectAckOutbound{
			Type:          wshub.EventUnsubscribed,
			SchemaVersion: wshub.EventSchemaVersion,
			ProjectID:     projectPtr,
		})
	default:
		w.writeWSError(connID, wshub.CodeUnknownType, "unknown type", &projectPtr)
	}
}

func (w *WSController) handleSubscribe(c echo.Context, connID, userID, projectID uuid.UUID, projectPtr string) {
	var (
		ctx      context.Context
		cancel   context.CancelFunc
		code     string
		internal bool
		err      error
	)
	ctx, cancel = context.WithTimeout(c.Request().Context(), wsDBOpTimeout)
	err = w.Messages.ConfirmWSSubscribe(ctx, userID, projectID, requestMeta(c))
	cancel()
	if err != nil {
		code, internal = apierr.MapWSSubscribe(err)
		if internal {
			w.writeWSError(connID, wshub.CodeInternal, "internal error", &projectPtr)
			return
		}
		msg := "not allowed to subscribe to this project"
		if code == wshub.CodeInvalidProjectID {
			msg = "project not found"
		}
		w.writeWSError(connID, code, msg, &projectPtr)
		return
	}
	w.Hub.Subscribe(connID, projectID)
	writeWSFrame(w.Hub, connID, wshub.ProjectAckOutbound{
		Type:          wshub.EventSubscribed,
		SchemaVersion: wshub.EventSchemaVersion,
		ProjectID:     projectPtr,
	})
}

func (w *WSController) handleTyping(connID, userID, projectID uuid.UUID, projectPtr string, isTyping *bool) {
	var (
		ctx      context.Context
		cancel   context.CancelFunc
		active   bool
		payload  []byte
		code     string
		internal bool
		err      error
	)
	if !w.Hub.AllowTyping(connID, projectID) {
		w.writeWSError(connID, wshub.CodeRateLimited, "typing rate limit exceeded", &projectPtr)
		return
	}
	active = true
	if isTyping != nil {
		active = *isTyping
	}
	ctx, cancel = context.WithTimeout(context.Background(), wsDBOpTimeout)
	err = w.Messages.CanSubscribeProject(ctx, projectID, userID)
	cancel()
	if err != nil {
		code, internal = apierr.MapWSSubscribe(err)
		if internal {
			w.writeWSError(connID, wshub.CodeInternal, "internal error", &projectPtr)
			return
		}
		w.writeWSError(connID, code, "not allowed", &projectPtr)
		return
	}
	payload, err = json.Marshal(wshub.TypingOutbound{
		Type:          wshub.EventTyping,
		SchemaVersion: wshub.EventSchemaVersion,
		ProjectID:     projectID,
		UserID:        userID,
		IsTyping:      active,
	})
	if err != nil {
		return
	}
	w.Hub.BroadcastToProjectExceptUser(projectID, payload, userID)
}

func (w *WSController) revalidateSubscriptions(connID, userID uuid.UUID) {
	var (
		projectID uuid.UUID
		ctx       context.Context
		cancel    context.CancelFunc
		err       error
	)
	for _, projectID = range w.Hub.ProjectIDs(connID) {
		ctx, cancel = context.WithTimeout(context.Background(), wsDBOpTimeout)
		err = w.Messages.CanSubscribeProject(ctx, projectID, userID)
		cancel()
		if err != nil {
			w.Hub.Unsubscribe(connID, projectID)
		}
	}
}

func (w *WSController) pingLoop(connID, userID uuid.UUID, done <-chan struct{}) {
	var ticker *time.Ticker
	ticker = time.NewTicker(wshub.PingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			w.revalidateSubscriptions(connID, userID)
			if !w.Hub.SendPing(connID) {
				w.Hub.RemoveConn(connID)
				return
			}
		}
	}
}

func (w *WSController) writeWSError(connID uuid.UUID, code, msg string, projectID *string) {
	writeWSFrame(w.Hub, connID, wshub.ErrorOutbound{
		Type:          wshub.EventError,
		SchemaVersion: wshub.EventSchemaVersion,
		Code:          code,
		Message:       msg,
		ProjectID:     projectID,
	})
}

func writeWSFrame[T any](hub wshub.Hub, connID uuid.UUID, v T) {
	var (
		b   []byte
		err error
	)
	b, err = json.Marshal(v)
	if err != nil {
		return
	}
	_ = hub.WriteText(connID, b)
}
