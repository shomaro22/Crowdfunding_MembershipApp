package usecase

import (
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
)

type UserView struct {
	ID            uuid.UUID
	Email         string
	Name          *string
	Role          entity.UserRole
	AccountStatus entity.AccountStatus
	SellerStatus  entity.SellerStatus
	AdminRole     *entity.AdminRole
	TOTPEnabled   bool
	AvatarURL     *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (v UserView) IsActive() bool {
	return v.AccountStatus == entity.AccountStatusActive
}

func UserViewFrom(u *entity.User) UserView {
	if u == nil {
		return UserView{}
	}
	return UserView{
		ID:            u.ID,
		Email:         u.Email,
		Name:          u.Name,
		Role:          u.Role,
		AccountStatus: u.AccountStatus,
		SellerStatus:  u.SellerStatus,
		AdminRole:     u.AdminRole,
		TOTPEnabled:   u.TOTPEnabled,
		AvatarURL:     u.AvatarURL,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
	}
}

func UserViewPtr(u *entity.User) *UserView {
	if u == nil {
		return nil
	}
	v := UserViewFrom(u)
	return &v
}

type ConnectAccountView struct {
	StripeAccountID  string
	ChargesEnabled   bool
	PayoutsEnabled   bool
	DetailsSubmitted bool
}

func ConnectAccountViewFrom(row *entity.ConnectAccount) *ConnectAccountView {
	if row == nil {
		return nil
	}
	return &ConnectAccountView{
		StripeAccountID:  row.StripeAccountID,
		ChargesEnabled:   row.ChargesEnabled,
		PayoutsEnabled:   row.PayoutsEnabled,
		DetailsSubmitted: row.DetailsSubmitted,
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
	Status       entity.RegStatus
	RejectReason *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func SellerRegItemFrom(r *entity.SellerReg) SellerRegItemView {
	if r == nil {
		return SellerRegItemView{}
	}
	return SellerRegItemView{
		ID:           r.ID,
		UserID:       r.UserID,
		FullName:     r.FullName,
		BirthDate:    r.BirthDate,
		Address:      r.Address,
		PhotoPresent: r.PhotoPresent,
		HasIDFront:   r.HasIDFront(),
		HasIDBack:    r.HasIDBack(),
		Status:       r.Status,
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
	DurationType    string
	StartDate       time.Time
	EndDate         *time.Time
	Status          string
	MainImageURL    *string
	DeleteRequested bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func ProjectItemFrom(p *entity.Project) ProjectItemView {
	if p == nil {
		return ProjectItemView{}
	}
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

func ProjectPlanFrom(plan entity.ProjectPlan) ProjectPlanView {
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

func ProjectPlanFromPtr(plan *entity.ProjectPlan) ProjectPlanView {
	if plan == nil {
		return ProjectPlanView{}
	}
	return ProjectPlanFrom(*plan)
}

func ProjectPlansFrom(plans []entity.ProjectPlan) []ProjectPlanView {
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

func ProjectImageFrom(img entity.ProjectImage) ProjectImageView {
	return ProjectImageView{
		ID:        img.ID,
		ProjectID: img.ProjectID,
		ImageURL:  img.ImageURL,
		IsMain:    img.IsMain,
		SortOrder: img.SortOrder,
		CreatedAt: img.CreatedAt,
	}
}

func ProjectImagesFrom(images []entity.ProjectImage) []ProjectImageView {
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

func AssembleProjectView(project *entity.Project, plans []entity.ProjectPlan, images []entity.ProjectImage, seller *UserView) ProjectView {
	return ProjectView{
		Project: ProjectItemFrom(project),
		Plans:   ProjectPlansFrom(plans),
		Images:  ProjectImagesFrom(images),
		Seller:  seller,
	}
}

func AssembleProjectViewPtr(project *entity.Project, plans []entity.ProjectPlan, images []entity.ProjectImage, seller *UserView) *ProjectView {
	v := AssembleProjectView(project, plans, images, seller)
	return &v
}

type SupporterItemView struct {
	ID                   uuid.UUID
	ProjectID            uuid.UUID
	PlanID               uuid.UUID
	UserID               uuid.UUID
	Status               string
	BillingMode          string
	CurrentPeriodStart   time.Time
	CurrentPeriodEnd     time.Time
	PendingPlanID        *uuid.UUID
	PlanChangeDate       *time.Time
}

func SupporterItemFrom(s *entity.ProjectSupporter) SupporterItemView {
	if s == nil {
		return SupporterItemView{}
	}
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
	Supporter   SupporterItemView
	Project     *ProjectItemView
	Plan        *ProjectPlanView
	PendingPlan *ProjectPlanView
}

func ProjectItemPtrFrom(p *entity.Project) *ProjectItemView {
	if p == nil {
		return nil
	}
	v := ProjectItemFrom(p)
	return &v
}

func ProjectPlanPtrFrom(plan *entity.ProjectPlan) *ProjectPlanView {
	if plan == nil {
		return nil
	}
	v := ProjectPlanFrom(*plan)
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

func PurchaseItemFrom(p *entity.Purchase) PurchaseItemView {
	if p == nil {
		return PurchaseItemView{}
	}
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
