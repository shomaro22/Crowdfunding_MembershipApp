package usecase

import (
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"
)

type PlanIn struct {
	MembershipType entity.MembershipType
	Price          int
	MaxSupporters  *int
	Benefits       string
}

type ImageIn struct {
	ImageURL string
	IsMain   bool
	Order    int
}

type CreateProjectIn struct {
	Title        string
	Description  string
	TargetAmount int64
	DurationType entity.DurationType
	StartDate    time.Time
	EndDate      *time.Time
	ImageURL     *string
	Images       []ImageIn
	Plans        []PlanIn
}

type UpdateProjectIn struct {
	Title        *string
	Description  *string
	TargetAmount *int64
	DurationType *entity.DurationType
	StartDate    *time.Time
	EndDate      *time.Time
	ImageURL     *string
	Images       *[]ImageIn
	IsReapply    bool
}

type ProjectView struct {
	Project *entity.Project
	Plans   []entity.ProjectPlan
	Images  []entity.ProjectImage
	Seller  *entity.User
}

type ProjectUseCase struct {
	Projects repository.ProjectRepo
	Users    repository.UserRepo
}
