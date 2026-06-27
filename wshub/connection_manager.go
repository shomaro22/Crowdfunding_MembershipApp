package wshub

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type ConnectionManager struct {
	mu          sync.RWMutex
	clients     map[uuid.UUID]*wsClient
	projectSubs map[uuid.UUID]map[uuid.UUID]struct{}
}

type wsClient struct {
	connID    uuid.UUID
	userID    uuid.UUID
	conn      *websocket.Conn
	writeMu   sync.Mutex
	projects  map[uuid.UUID]struct{}
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		clients:     make(map[uuid.UUID]*wsClient),
		projectSubs: make(map[uuid.UUID]map[uuid.UUID]struct{}),
	}
}

func (m *ConnectionManager) Register(userID uuid.UUID, conn *websocket.Conn) uuid.UUID {
	var (
		id uuid.UUID
		c  *wsClient
	)
	id = uuid.New()
	c = &wsClient{
		connID:   id,
		userID:   userID,
		conn:     conn,
		projects: make(map[uuid.UUID]struct{}),
	}
	m.mu.Lock()
	m.clients[id] = c
	m.mu.Unlock()
	return id
}

func (m *ConnectionManager) removeConnLocked(connID uuid.UUID) *websocket.Conn {
	var (
		c      *wsClient
		ok     bool
		wsConn *websocket.Conn
		pid    uuid.UUID
	)
	c, ok = m.clients[connID]
	if !ok {
		return nil
	}
	wsConn = c.conn
	for pid = range c.projects {
		if subs, ok := m.projectSubs[pid]; ok {
			delete(subs, connID)
			if len(subs) == 0 {
				delete(m.projectSubs, pid)
			}
		}
	}
	delete(m.clients, connID)
	return wsConn
}

func (m *ConnectionManager) RemoveConn(connID uuid.UUID) {
	var wsConn *websocket.Conn
	m.mu.Lock()
	wsConn = m.removeConnLocked(connID)
	m.mu.Unlock()
	if wsConn != nil {
		_ = wsConn.Close()
	}
}

func (m *ConnectionManager) Subscribe(connID, projectID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.clients[connID]
	if !ok {
		return
	}
	c.projects[projectID] = struct{}{}
	if m.projectSubs[projectID] == nil {
		m.projectSubs[projectID] = make(map[uuid.UUID]struct{})
	}
	m.projectSubs[projectID][connID] = struct{}{}
}

func (m *ConnectionManager) Unsubscribe(connID, projectID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.clients[connID]
	if !ok {
		return
	}
	delete(c.projects, projectID)
	if subs, ok := m.projectSubs[projectID]; ok {
		delete(subs, connID)
		if len(subs) == 0 {
			delete(m.projectSubs, projectID)
		}
	}
}

func (m *ConnectionManager) BroadcastToProject(projectID uuid.UUID, payload []byte) {
	var (
		list []*wsClient
		c    *wsClient
		cid  uuid.UUID
		ok   bool
	)
	m.mu.RLock()
	subs := m.projectSubs[projectID]
	if len(subs) == 0 {
		m.mu.RUnlock()
		return
	}
	list = make([]*wsClient, 0, len(subs))
	for cid = range subs {
		if c, ok = m.clients[cid]; ok {
			list = append(list, c)
		}
	}
	m.mu.RUnlock()
	for _, c = range list {
		c.writeMu.Lock()
		c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		_ = c.conn.WriteMessage(websocket.TextMessage, payload)
		c.writeMu.Unlock()
	}
}

func (m *ConnectionManager) BroadcastToProjectExceptUser(projectID uuid.UUID, payload []byte, excludeUserID uuid.UUID) {
	var (
		list []*wsClient
		c    *wsClient
		cid  uuid.UUID
		ok   bool
	)
	m.mu.RLock()
	subs := m.projectSubs[projectID]
	if len(subs) == 0 {
		m.mu.RUnlock()
		return
	}
	list = make([]*wsClient, 0, len(subs))
	for cid = range subs {
		if c, ok = m.clients[cid]; ok && c.userID != excludeUserID {
			list = append(list, c)
		}
	}
	m.mu.RUnlock()
	for _, c = range list {
		c.writeMu.Lock()
		c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		_ = c.conn.WriteMessage(websocket.TextMessage, payload)
		c.writeMu.Unlock()
	}
}

func (m *ConnectionManager) SendPing(connID uuid.UUID) bool {
	var (
		c   *wsClient
		ok  bool
		err error
	)
	m.mu.RLock()
	c, ok = m.clients[connID]
	m.mu.RUnlock()
	if !ok {
		return false
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	err = c.conn.WriteMessage(websocket.PingMessage, nil)
	return err == nil
}

func (m *ConnectionManager) WriteText(connID uuid.UUID, payload []byte) bool {
	var (
		c   *wsClient
		ok  bool
		err error
	)
	m.mu.RLock()
	c, ok = m.clients[connID]
	m.mu.RUnlock()
	if !ok {
		return false
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	err = c.conn.WriteMessage(websocket.TextMessage, payload)
	return err == nil
}
