package usecase

import (
	"encoding/json"
	"time"

	"crowdfund-app/model"

	"github.com/google/uuid"
)

type MessageItemView struct {
	ID         uuid.UUID
	ProjectID  uuid.UUID
	SenderID   uuid.UUID
	ReceiverID uuid.UUID
	Content    string
	IsRead     bool
	ReadAt     *time.Time
	CreatedAt  time.Time
	Sender     *UserView
	Receiver   *UserView
}

type MessageThreadView struct {
	ProjectID    uuid.UUID
	ProjectTitle string
	Partner      UserView
	Messages     []MessageItemView
	UnreadCount  int
}

type NotificationsView struct {
	UnreadMessages int64
}

type ListThreadsQuery struct {
	Page              int
	PerPage           int
	MessagesPerThread int
}

type ListThreadsResult struct {
	Threads []MessageThreadView
	Total   int64
	Page    int
	PerPage int
}

func MessageItemFrom(msg model.Message) MessageItemView {
	return MessageItemView{
		ID:         msg.ID,
		ProjectID:  msg.ProjectID,
		SenderID:   msg.SenderID,
		ReceiverID: msg.ReceiverID,
		Content:    msg.Content,
		IsRead:     msg.IsRead,
		ReadAt:     msg.ReadAt,
		CreatedAt:  msg.CreatedAt,
	}
}

func MessageItemWithUsers(msg model.Message, sender, receiver *UserView) MessageItemView {
	var out MessageItemView
	out = MessageItemFrom(msg)
	out.Sender = sender
	out.Receiver = receiver
	return out
}

type ActivityLogView struct {
	ID         uuid.UUID
	UserID     *uuid.UUID
	Action     string
	Resource   *string
	ResourceID *string
	IPAddress  *string
	UserAgent  *string
	RequestID  *string
	Metadata   json.RawMessage
	CreatedAt  time.Time
	User       *UserView
}

type ActivityLogsResult struct {
	Logs   []ActivityLogView
	Total  int64
	Limit  int
	Offset int
}

type AuditLogView struct {
	ID         uuid.UUID
	AdminID    uuid.UUID
	AdminEmail string
	Action     string
	Resource   string
	ResourceID string
	Details    *string
	Reason     *string
	CreatedAt  time.Time
}

type AuditLogsResult struct {
	Logs   []AuditLogView
	Total  int64
	Limit  int
	Offset int
}

type BatchRunsResult struct {
	Runs  []BatchRunView
	Total int64
	Limit int
}

type SecurityIPCountView struct {
	IP    string
	Count int64
}

type SecurityEventsView struct {
	LoginFailedCount int64
	RateLimitedCount int64
	TopIPs           []SecurityIPCountView
	From             time.Time
	To               time.Time
}

func ActivityLogViewFrom(row model.ActivityLog) ActivityLogView {
	return ActivityLogView{
		ID:         row.ID,
		UserID:     row.UserID,
		Action:     string(row.Action),
		Resource:   row.Resource,
		ResourceID: row.ResourceID,
		IPAddress:  row.IPAddress,
		UserAgent:  row.UserAgent,
		RequestID:  row.RequestID,
		Metadata:   json.RawMessage(row.Metadata),
		CreatedAt:  row.CreatedAt,
	}
}

func AuditLogViewFrom(row model.AuditLog, adminEmail string) AuditLogView {
	return AuditLogView{
		ID:         row.ID,
		AdminID:    row.AdminID,
		AdminEmail: adminEmail,
		Action:     row.Action,
		Resource:   row.Resource,
		ResourceID: row.ResourceID,
		Details:    row.Details,
		Reason:     row.Reason,
		CreatedAt:  row.CreatedAt,
	}
}
