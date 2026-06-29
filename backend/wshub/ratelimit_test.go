package wshub_test

import (
	"testing"
	"time"

	"crowdfund-app/wshub"

	"github.com/google/uuid"
)

func TestAllowTypingRateLimit(t *testing.T) {
	var (
		hub       *wshub.ConnectionManager
		connID    uuid.UUID
		projectID uuid.UUID
	)
	hub = wshub.NewConnectionManager()
	projectID = uuid.New()
	connID = hub.Register(uuid.New(), nil)
	if !hub.AllowTyping(connID, projectID) {
		t.Fatal("first typing should be allowed")
	}
	if hub.AllowTyping(connID, projectID) {
		t.Fatal("second typing within interval should be blocked")
	}
	time.Sleep(wshub.TypingMinInterval + 10*time.Millisecond)
	if !hub.AllowTyping(connID, projectID) {
		t.Fatal("typing after interval should be allowed")
	}
}

func TestBroadcastToProjectAsyncDoesNotBlock(t *testing.T) {
	var (
		hub       *wshub.ConnectionManager
		projectID uuid.UUID
		done      chan struct{}
	)
	hub = wshub.NewConnectionManager()
	projectID = uuid.New()
	done = make(chan struct{})
	go func() {
		hub.BroadcastToProjectAsync(projectID, []byte(`{"type":"ping"}`))
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("async broadcast call blocked")
	}
}
