package usecase

import (
	"encoding/json"
	"time"

	"crowdfund-app/model"

	"github.com/google/uuid"
)

type UserView struct {
	ID            uuid.UUID
	Email         string
	Name          *string
	Role          string
	AccountStatus string
	SellerStatus  string
	AdminRole     *string
	TOTPEnabled   bool
	AvatarURL     *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (v UserView) IsActive() bool {
	return v.AccountStatus == string(model.AccountStatusActive)
}

func adminRoleString(r *model.AdminRole) *string {
	if r == nil {
		return nil
	}
	s := string(*r)
	return &s
}

func UserViewFrom(u model.User) UserView {
	return UserView{
		ID:            u.ID,
		Email:         u.Email,
		Name:          u.Name,
		Role:          string(u.Role),
		AccountStatus: string(u.AccountStatus),
		SellerStatus:  string(u.SellerStatus),
		AdminRole:     adminRoleString(u.AdminRole),
		TOTPEnabled:   u.TOTPEnabled,
		AvatarURL:     u.AvatarURL,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
	}
}

func UserViewPtr(u model.User) *UserView {
	v := UserViewFrom(u)
	return &v
}

type ConnectAccountView struct {
	StripeAccountID  string
	ChargesEnabled   bool
	PayoutsEnabled   bool
	DetailsSubmitted bool
}

func ConnectAccountViewFrom(row model.ConnectAccount) *ConnectAccountView {
	return &ConnectAccountView{
		StripeAccountID:  row.StripeAccountID,
		ChargesEnabled:   row.ChargesEnabled,
		PayoutsEnabled:   row.PayoutsEnabled,
		DetailsSubmitted: row.DetailsSubmitted,
	}
}

type BatchRunView struct {
	ID           uuid.UUID
	JobName      string
	Status       string
	StartedAt    time.Time
	FinishedAt   *time.Time
	Processed    int
	Failed       int
	ErrorMessage *string
	TriggeredBy  string
	DryRun       bool
}

func BatchRunViewFrom(r *model.BatchRun) *BatchRunView {
	if r == nil {
		return nil
	}
	return &BatchRunView{
		ID:           r.ID,
		JobName:      r.JobName,
		Status:       string(r.Status),
		StartedAt:    r.StartedAt,
		FinishedAt:   r.FinishedAt,
		Processed:    r.Processed,
		Failed:       r.Failed,
		ErrorMessage: r.ErrorMessage,
		TriggeredBy:  r.TriggeredBy,
		DryRun:       r.DryRun,
	}
}

type SellerRegItemView struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	FullName     string
	BirthDate    time.Time
	Address      string
	PhotoPresent bool
	HasIDFront   bool
	HasIDBack    bool
	Status       string
	RejectReason *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func SellerRegItemFrom(r model.SellerReg) SellerRegItemView {
	return SellerRegItemView{
		ID:           r.ID,
		UserID:       r.UserID,
		FullName:     r.FullName,
		BirthDate:    r.BirthDate,
		Address:      r.Address,
		PhotoPresent: r.PhotoPresent,
		HasIDFront:   r.HasIDFront(),
		HasIDBack:    r.HasIDBack(),
		Status:       string(r.Status),
		RejectReason: r.RejectReason,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

type ProjectItemView struct {
	ID              uuid.UUID
	SellerID        uuid.UUID
	Title           string
	Description     string
	TargetAmount    int64
	CurrentAmount   int64
	SupporterCount  int
	FavoriteCount   int64
	DurationType    string
	StartDate       time.Time
	EndDate         *time.Time
	Status          string
	MainImageURL    *string
	DeleteRequested bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func ProjectItemFrom(p model.Project) ProjectItemView {
	return ProjectItemView{
		ID:              p.ID,
		SellerID:        p.SellerID,
		Title:           p.Title,
		Description:     p.Description,
		TargetAmount:    p.TargetAmount,
		CurrentAmount:   p.CurrentAmount,
		SupporterCount:  p.SupporterCount,
		DurationType:    string(p.DurationType),
		StartDate:       p.StartDate,
		EndDate:         p.EndDate,
		Status:          string(p.Status),
		MainImageURL:    p.MainImageURL,
		DeleteRequested: p.DeleteRequested,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
}

type ProjectPlanView struct {
	ID             uuid.UUID
	ProjectID      uuid.UUID
	MembershipType string
	Price          int
	MaxSupporters  *int
	Benefits       string
	IsActive       bool
	SupporterCount int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func ProjectPlanFrom(plan model.ProjectPlan) ProjectPlanView {
	return ProjectPlanView{
		ID:             plan.ID,
		ProjectID:      plan.ProjectID,
		MembershipType: string(plan.MembershipType),
		Price:          plan.Price,
		MaxSupporters:  plan.MaxSupporters,
		Benefits:       plan.Benefits,
		IsActive:       plan.IsActive,
		SupporterCount: plan.SupporterCount,
		CreatedAt:      plan.CreatedAt,
		UpdatedAt:      plan.UpdatedAt,
	}
}

func ProjectPlansFrom(plans []model.ProjectPlan) []ProjectPlanView {
	var (
		out = make([]ProjectPlanView, len(plans))
		i   int
	)
	for i = range plans {
		out[i] = ProjectPlanFrom(plans[i])
	}
	return out
}

type ProjectImageView struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	ImageURL  string
	IsMain    bool
	SortOrder int
	CreatedAt time.Time
}

func ProjectImageFrom(img model.ProjectImage) ProjectImageView {
	return ProjectImageView{
		ID:        img.ID,
		ProjectID: img.ProjectID,
		ImageURL:  img.ImageURL,
		IsMain:    img.IsMain,
		SortOrder: img.SortOrder,
		CreatedAt: img.CreatedAt,
	}
}

func ProjectImagesFrom(images []model.ProjectImage) []ProjectImageView {
	var (
		out = make([]ProjectImageView, len(images))
		i   int
	)
	for i = range images {
		out[i] = ProjectImageFrom(images[i])
	}
	return out
}

type ProjectView struct {
	Project ProjectItemView
	Plans   []ProjectPlanView
	Images  []ProjectImageView
	Seller  *UserView
}

func AssembleProjectView(project model.Project, plans []model.ProjectPlan, images []model.ProjectImage, seller *UserView) ProjectView {
	return ProjectView{
		Project: ProjectItemFrom(project),
		Plans:   ProjectPlansFrom(plans),
		Images:  ProjectImagesFrom(images),
		Seller:  seller,
	}
}

func AssembleProjectViewPtr(project model.Project, plans []model.ProjectPlan, images []model.ProjectImage, seller *UserView) *ProjectView {
	v := AssembleProjectView(project, plans, images, seller)
	return &v
}

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

type SupporterItemView struct {
	ID                 uuid.UUID
	ProjectID          uuid.UUID
	PlanID             uuid.UUID
	UserID             uuid.UUID
	Status             string
	BillingMode        string
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time
	PendingPlanID      *uuid.UUID
	PlanChangeDate     *time.Time
}

func SupporterItemFrom(s model.ProjectSupporter) SupporterItemView {
	return SupporterItemView{
		ID:                 s.ID,
		ProjectID:          s.ProjectID,
		PlanID:             s.PlanID,
		UserID:             s.UserID,
		Status:             string(s.Status),
		BillingMode:        string(s.BillingMode),
		CurrentPeriodStart: s.CurrentPeriodStart,
		CurrentPeriodEnd:   s.CurrentPeriodEnd,
		PendingPlanID:      s.PendingPlanID,
		PlanChangeDate:     s.PlanChangeDate,
	}
}

type SupporterView struct {
	Supporter SupporterItemView
	Project   *ProjectItemView
	Plan      *ProjectPlanView
}

func ProjectItemPtrFrom(p model.Project) *ProjectItemView {
	v := ProjectItemFrom(p)
	return &v
}

func ProjectPlanPtrFrom(plan model.ProjectPlan) *ProjectPlanView {
	v := ProjectPlanFrom(plan)
	return &v
}

type PurchaseItemView struct {
	ID            uuid.UUID
	ProjectID     uuid.UUID
	SupporterID   *uuid.UUID
	BuyerID       uuid.UUID
	Quantity      int
	TotalPrice    int
	Status        string
	PaymentMethod string
	CreatedAt     time.Time
}

func PurchaseItemFrom(p model.Purchase) PurchaseItemView {
	return PurchaseItemView{
		ID:            p.ID,
		ProjectID:     p.ProjectID,
		SupporterID:   p.SupporterID,
		BuyerID:       p.BuyerID,
		Quantity:      p.Quantity,
		TotalPrice:    p.TotalPrice,
		Status:        string(p.Status),
		PaymentMethod: string(p.PaymentMethod),
		CreatedAt:     p.CreatedAt,
	}
}

type PurchaseView struct {
	Purchase PurchaseItemView
	Project  *ProjectItemView
	Buyer    *UserView
}
