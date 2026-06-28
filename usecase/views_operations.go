package usecase

import (
	"encoding/json"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
)

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
	LoginFailedCount  int64
	RateLimitedCount  int64
	TopIPs            []SecurityIPCountView
	From              time.Time
	To                time.Time
}

func ActivityLogViewFrom(row entity.ActivityLog) ActivityLogView {
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

func AuditLogViewFrom(row entity.AuditLog, adminEmail string) AuditLogView {
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
