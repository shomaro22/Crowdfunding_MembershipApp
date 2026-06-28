package controller

import (
	"encoding/json"
	"time"

	"crowdfund-app/auth"
	"crowdfund-app/usecase"
)

type usersPageJSON struct {
	Users []UserWire `json:"users"`
	Total int64      `json:"total"`
}

type sellerRegsPageJSON struct {
	Registrations []sellerRegJSON `json:"registrations"`
	Total         int64           `json:"total"`
}

type sellerRegViewJSON struct {
	sellerRegJSON
	User     UserWire `json:"user"`
	FrontURL *string  `json:"identity_document_front_url,omitempty"`
	BackURL  *string  `json:"identity_document_back_url,omitempty"`
}

type adminStatsWire struct {
	TotalUsers     int64      `json:"total_users"`
	TotalProjects  int64      `json:"total_projects"`
	TotalPurchases int64      `json:"total_purchases"`
	TotalRevenue   int64      `json:"total_revenue"`
	RecentUsers    []UserWire `json:"recent_users"`
}

type adminUpdateUserJSON struct {
	Email    *string `json:"email"`
	Name     *string `json:"name"`
	Role     *string `json:"role"`
	IsActive *bool   `json:"is_active"`
}

type adminPatchPurchaseJSON struct {
	Status string `json:"status"`
}

func sellerRegViewToJSON(apiBase string, d *usecase.SellerRegView) sellerRegViewJSON {
	var (
		out sellerRegViewJSON
		url string
	)
	out = sellerRegViewJSON{
		sellerRegJSON: sellerRegToJSON(d.Registration),
		User:          UserWireFrom(d.User),
	}
	if d.FrontAccessToken != nil {
		url = auth.KYCAccessURL(apiBase, *d.FrontAccessToken)
		out.FrontURL = &url
	}
	if d.BackAccessToken != nil {
		url = auth.KYCAccessURL(apiBase, *d.BackAccessToken)
		out.BackURL = &url
	}
	return out
}

type ActivityLogWire struct {
	ID         string          `json:"id"`
	UserID     *string         `json:"user_id,omitempty"`
	Action     string          `json:"action"`
	Resource   *string         `json:"resource,omitempty"`
	ResourceID *string         `json:"resource_id,omitempty"`
	IPAddress  *string         `json:"ip_address,omitempty"`
	UserAgent  *string         `json:"user_agent,omitempty"`
	RequestID  *string         `json:"request_id,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
	CreatedAt  string          `json:"created_at"`
	User       *UserWire       `json:"user,omitempty"`
}

type ListActivityLogsWire struct {
	Logs   []ActivityLogWire `json:"logs"`
	Total  int64             `json:"total"`
	Limit  int               `json:"limit,omitempty"`
	Offset int               `json:"offset,omitempty"`
}

type AuditLogWire struct {
	ID         string  `json:"id"`
	AdminID    string  `json:"admin_id"`
	AdminEmail string  `json:"admin_email"`
	Action     string  `json:"action"`
	Resource   string  `json:"resource"`
	ResourceID string  `json:"resource_id"`
	Details    *string `json:"details,omitempty"`
	Reason     *string `json:"reason,omitempty"`
	CreatedAt  string  `json:"created_at"`
}

type ListAuditLogsWire struct {
	Logs   []AuditLogWire `json:"logs"`
	Total  int64          `json:"total"`
	Limit  int            `json:"limit,omitempty"`
	Offset int            `json:"offset,omitempty"`
}

type BatchJobRunWire struct {
	ID           string  `json:"id"`
	JobName      string  `json:"job_name"`
	Status       string  `json:"status"`
	StartedAt    string  `json:"started_at"`
	FinishedAt   *string `json:"finished_at,omitempty"`
	Processed    int     `json:"processed"`
	Failed       int     `json:"failed"`
	ErrorMessage *string `json:"error_message,omitempty"`
	TriggeredBy  string  `json:"triggered_by"`
}

type ListBatchJobRunsWire struct {
	Runs  []BatchJobRunWire `json:"runs"`
	Total int64             `json:"total"`
	Limit int               `json:"limit,omitempty"`
}

type SecurityIPCountWire struct {
	IP    string `json:"ip"`
	Count int64  `json:"count"`
}

type SecurityEventsWire struct {
	LoginFailedCount int64                 `json:"login_failed_count"`
	RateLimitedCount int64                 `json:"rate_limited_count"`
	TopIPs           []SecurityIPCountWire `json:"top_ips"`
	From             string                `json:"from,omitempty"`
	To               string                `json:"to,omitempty"`
}

func ActivityLogWireFrom(v usecase.ActivityLogView) ActivityLogWire {
	var (
		out ActivityLogWire
		uid *string
		uw  UserWire
	)
	out = ActivityLogWire{
		ID:         v.ID.String(),
		Action:     v.Action,
		Resource:   v.Resource,
		ResourceID: v.ResourceID,
		IPAddress:  v.IPAddress,
		UserAgent:  v.UserAgent,
		RequestID:  v.RequestID,
		Metadata:   v.Metadata,
		CreatedAt:  v.CreatedAt.UTC().Format(time.RFC3339),
	}
	if v.UserID != nil {
		s := v.UserID.String()
		uid = &s
		out.UserID = uid
	}
	if v.User != nil {
		uw = UserWireFrom(*v.User)
		out.User = &uw
	}
	return out
}

func ListActivityLogsWireFrom(r usecase.ActivityLogsResult) ListActivityLogsWire {
	var (
		out ListActivityLogsWire
		i   int
	)
	out = ListActivityLogsWire{
		Total:  r.Total,
		Limit:  r.Limit,
		Offset: r.Offset,
		Logs:   make([]ActivityLogWire, len(r.Logs)),
	}
	for i = range r.Logs {
		out.Logs[i] = ActivityLogWireFrom(r.Logs[i])
	}
	return out
}

func AuditLogWireFrom(v usecase.AuditLogView) AuditLogWire {
	return AuditLogWire{
		ID:         v.ID.String(),
		AdminID:    v.AdminID.String(),
		AdminEmail: v.AdminEmail,
		Action:     v.Action,
		Resource:   v.Resource,
		ResourceID: v.ResourceID,
		Details:    v.Details,
		Reason:     v.Reason,
		CreatedAt:  v.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func ListAuditLogsWireFrom(r usecase.AuditLogsResult) ListAuditLogsWire {
	var (
		out ListAuditLogsWire
		i   int
	)
	out = ListAuditLogsWire{
		Total:  r.Total,
		Limit:  r.Limit,
		Offset: r.Offset,
		Logs:   make([]AuditLogWire, len(r.Logs)),
	}
	for i = range r.Logs {
		out.Logs[i] = AuditLogWireFrom(r.Logs[i])
	}
	return out
}

func BatchJobRunWireFrom(v usecase.BatchRunView) BatchJobRunWire {
	var (
		out        BatchJobRunWire
		finishedAt *string
		s          string
	)
	out = BatchJobRunWire{
		ID:           v.ID.String(),
		JobName:      v.JobName,
		Status:       v.Status,
		StartedAt:    v.StartedAt.UTC().Format(time.RFC3339),
		Processed:    v.Processed,
		Failed:       v.Failed,
		ErrorMessage: v.ErrorMessage,
		TriggeredBy:  v.TriggeredBy,
	}
	if v.FinishedAt != nil {
		s = v.FinishedAt.UTC().Format(time.RFC3339)
		finishedAt = &s
		out.FinishedAt = finishedAt
	}
	return out
}

func ListBatchJobRunsWireFrom(r usecase.BatchRunsResult) ListBatchJobRunsWire {
	var (
		out ListBatchJobRunsWire
		i   int
	)
	out = ListBatchJobRunsWire{
		Total: r.Total,
		Limit: r.Limit,
		Runs:  make([]BatchJobRunWire, len(r.Runs)),
	}
	for i = range r.Runs {
		out.Runs[i] = BatchJobRunWireFrom(r.Runs[i])
	}
	return out
}

func SecurityEventsWireFrom(v usecase.SecurityEventsView) SecurityEventsWire {
	var (
		out SecurityEventsWire
		i   int
	)
	out = SecurityEventsWire{
		LoginFailedCount: v.LoginFailedCount,
		RateLimitedCount: v.RateLimitedCount,
		From:             v.From.UTC().Format(time.RFC3339),
		To:               v.To.UTC().Format(time.RFC3339),
		TopIPs:           make([]SecurityIPCountWire, len(v.TopIPs)),
	}
	for i = range v.TopIPs {
		out.TopIPs[i] = SecurityIPCountWire{
			IP:    v.TopIPs[i].IP,
			Count: v.TopIPs[i].Count,
		}
	}
	return out
}
