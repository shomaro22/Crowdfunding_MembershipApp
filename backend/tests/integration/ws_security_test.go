//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"crowdfund-app/model"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func TestWSInvalidTokenRejected(t *testing.T) {
	var (
		e     *echo.Echo
		srv   *httptest.Server
		wsURL string
		err   error
	)
	e = setupApp(t)
	srv = httptest.NewServer(e)
	defer srv.Close()

	wsURL = strings.Replace(srv.URL, "http", "ws", 1) + "/ws?ws_token=invalid-token"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected dial failure for invalid ws token")
	}
	if resp != nil && resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", resp.StatusCode)
	}
}

func TestWSSubscribeForbiddenWithoutMembership(t *testing.T) {
	var (
		e        *echo.Echo
		gdb      *gorm.DB
		seller   *model.User
		buyer    *model.User
		project  *model.Project
		srv      *httptest.Server
		buyerTok string
		rec      *httptest.ResponseRecorder
		wsPlain  string
		wsConn   *websocket.Conn
		wsURL    string
		err      error
	)
	e = setupApp(t)
	gdb = testDB(t)
	seller = seedUser(t, gdb, seedOpts{Role: model.UserRoleSeller, TOTPEnabled: true})
	buyer = seedUser(t, gdb, seedOpts{Role: model.UserRoleBuyer})
	project = seedProject(t, gdb, seller.ID, model.ProjectStatusActive)

	buyerTok, err = issueToken(buyer, true)
	if err != nil {
		t.Fatal(err)
	}
	rec = httpPOST(e, "/ws/token", buyerTok, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("ws token status=%d body=%s", rec.Code, rec.Body.String())
	}
	wsPlain = jsonStringField(t, rec, "ws_token")

	srv = httptest.NewServer(e)
	defer srv.Close()
	wsURL = strings.Replace(srv.URL, "http", "ws", 1) + "/ws?ws_token=" + wsPlain
	wsConn, _, err = websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer wsConn.Close()

	wsConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, _, err = wsConn.ReadMessage()
	if err != nil {
		t.Fatalf("session_ready: %v", err)
	}

	subscribe := fmt.Sprintf(`{"type":"subscribe","project_id":%q}`, project.ID)
	err = wsConn.WriteMessage(websocket.TextMessage, []byte(subscribe))
	if err != nil {
		t.Fatal(err)
	}
	wsConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, raw, err := wsConn.ReadMessage()
	if err != nil {
		t.Fatalf("subscribe response: %v", err)
	}
	msg := string(raw)
	if !strings.Contains(msg, "FORBIDDEN_NOT_MEMBER") && !strings.Contains(msg, "error") {
		t.Fatalf("expected forbidden subscribe, got: %s", msg)
	}
}

func TestWSUnknownProjectSubscribe(t *testing.T) {
	var (
		e        *echo.Echo
		gdb      *gorm.DB
		buyer    *model.User
		srv      *httptest.Server
		buyerTok string
		rec      *httptest.ResponseRecorder
		wsPlain  string
		wsConn   *websocket.Conn
		wsURL    string
		err      error
	)
	e = setupApp(t)
	gdb = testDB(t)
	buyer = seedUser(t, gdb, seedOpts{Role: model.UserRoleBuyer})
	buyerTok, err = issueToken(buyer, true)
	if err != nil {
		t.Fatal(err)
	}
	rec = httpPOST(e, "/ws/token", buyerTok, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("ws token status=%d", rec.Code)
	}
	wsPlain = jsonStringField(t, rec, "ws_token")

	srv = httptest.NewServer(e)
	defer srv.Close()
	wsURL = strings.Replace(srv.URL, "http", "ws", 1) + "/ws?ws_token=" + wsPlain
	wsConn, _, err = websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer wsConn.Close()

	wsConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, _, err = wsConn.ReadMessage()
	if err != nil {
		t.Fatalf("session_ready: %v", err)
	}

	unknownID := uuid.New()
	subscribe := fmt.Sprintf(`{"type":"subscribe","project_id":%q}`, unknownID)
	err = wsConn.WriteMessage(websocket.TextMessage, []byte(subscribe))
	if err != nil {
		t.Fatal(err)
	}
	wsConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, raw, err := wsConn.ReadMessage()
	if err != nil {
		t.Fatalf("subscribe response: %v", err)
	}
	msg := string(raw)
	if !strings.Contains(msg, "INVALID_PROJECT_ID") && !strings.Contains(msg, "error") {
		t.Fatalf("expected invalid project, got: %s", msg)
	}
}
