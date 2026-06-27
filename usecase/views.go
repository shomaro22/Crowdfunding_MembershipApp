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

func ProjectPlanFrom(pl entity.ProjectPlan) ProjectPlanView {
	return ProjectPlanView{
		ID:             pl.ID,
		ProjectID:      pl.ProjectID,
		MembershipType: string(pl.MembershipType),
		Price:          pl.Price,
		MaxSupporters:  pl.MaxSupporters,
		Benefits:       pl.Benefits,
		IsActive:       pl.IsActive,
		SupporterCount: pl.SupporterCount,
		CreatedAt:      pl.CreatedAt,
		UpdatedAt:      pl.UpdatedAt,
	}
}

func ProjectPlanFromPtr(pl *entity.ProjectPlan) ProjectPlanView {
	if pl == nil {
		return ProjectPlanView{}
	}
	return ProjectPlanFrom(*pl)
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
