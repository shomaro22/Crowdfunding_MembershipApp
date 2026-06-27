package wshub_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"crowdfund-app/wshub"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func TestConnectionManagerSubscribeBroadcast(t *testing.T) {
	var (
		hub       *wshub.ConnectionManager
		srv       *httptest.Server
		projectID uuid.UUID
		userID    uuid.UUID
		connID    uuid.UUID
		payload   []byte
	)
	hub = wshub.NewConnectionManager()
	projectID = uuid.New()
	userID = uuid.New()
	payload = []byte(`{"type":"message_created"}`)

	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			upgrader websocket.Upgrader
			conn     *websocket.Conn
			err      error
		)
		upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
		conn, err = upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		connID = hub.Register(userID, conn)
		hub.Subscribe(connID, projectID)
		hub.BroadcastToProject(projectID, payload)
		time.Sleep(50 * time.Millisecond)
		hub.RemoveConn(connID)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(msg) != string(payload) {
		t.Fatalf("payload=%q want %q", msg, payload)
	}
}
