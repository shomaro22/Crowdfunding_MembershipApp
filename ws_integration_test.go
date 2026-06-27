//go:build integration

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/wshub"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func TestWSMessageCreatedBroadcast(t *testing.T) {
	var (
		e         *echo.Echo
		gdb       *gorm.DB
		fx        paymentFixture
		srv       *httptest.Server
		buyerTok  string
		sellerTok string
		wsPlain   string
		rec       *httptest.ResponseRecorder
		wsConn    *websocket.Conn
		msg       string
		now       time.Time
		err       error
	)
	e, _ = setupPaymentApp(t)
	gdb = openSecurityDB(t)
	fx = seedPaymentFixture(t, gdb)
	now = time.Now().UTC()
	err = gdb.Create(&entity.ProjectSupporter{
		ID:                 uuid.New(),
		ProjectID:          fx.project.ID,
		UserID:             fx.buyer.ID,
		PlanID:             fx.plan.ID,
		Status:             entity.SupporterStatusActive,
		BillingMode:        entity.BillingModeKonbini,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 0, 30),
		CreatedAt:          now,
		UpdatedAt:          now,
	}).Error
	if err != nil {
		t.Fatal(err)
	}

	buyerTok, err = issueSecurityToken(fx.buyer, true)
	if err != nil {
		t.Fatal(err)
	}
	sellerTok, err = issueSecurityToken(fx.seller, true)
	if err != nil {
		t.Fatal(err)
	}

	rec = securityPOST(e, "/ws/token", buyerTok, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("ws token status=%d body=%s", rec.Code, rec.Body.String())
	}
	wsPlain = jsonStringField(t, rec, "ws_token")
	if wsPlain == "" {
		t.Fatal("missing ws_token")
	}

	srv = httptest.NewServer(e)
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http", "ws", 1) + "/ws?ws_token=" + wsPlain
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

	subscribe := fmt.Sprintf(`{"type":"subscribe","project_id":%q}`, fx.project.ID)
	err = wsConn.WriteMessage(websocket.TextMessage, []byte(subscribe))
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = wsConn.ReadMessage()
	if err != nil {
		t.Fatalf("subscribed ack: %v", err)
	}

	rec = securityPOST(e, "/messages", sellerTok, fmt.Sprintf(
		`{"project_id":%q,"receiver_id":%q,"content":"hello ws"}`,
		fx.project.ID, fx.buyer.ID,
	))
	if rec.Code != http.StatusCreated {
		t.Fatalf("send status=%d body=%s", rec.Code, rec.Body.String())
	}

	wsConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, raw, err := wsConn.ReadMessage()
	if err != nil {
		t.Fatalf("message_created: %v", err)
	}
	msg = string(raw)
	if !strings.Contains(msg, wshub.EventMessageCreated) {
		t.Fatalf("unexpected ws payload: %s", msg)
	}
	if !strings.Contains(msg, "hello ws") {
		t.Fatalf("missing content in payload: %s", msg)
	}
}
