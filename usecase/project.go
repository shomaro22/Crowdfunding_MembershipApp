package usecase

import (
	"context"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"

	"github.com/google/uuid"
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

func (p *ProjectUseCase) enrichList(ctx context.Context, rows []entity.Project, total int64) ([]ProjectView, int64, error) {
	var (
		out         []ProjectView
		sellerCache map[uuid.UUID]*entity.User
		i           int
		proj        entity.Project
		seller      *entity.User
		ok          bool
		u           *entity.User
		plans       []entity.ProjectPlan
		images      []entity.ProjectImage
		err         error
		pcopy       entity.Project
	)

	out = make([]ProjectView, 0, len(rows))
	sellerCache = map[uuid.UUID]*entity.User{}
	for i = range rows {
		proj = rows[i]
		seller, ok = sellerCache[proj.SellerID]
		if !ok {
			u, err = p.Users.FindByID(ctx, proj.SellerID)
			if err != nil {
				return nil, 0, err
			}
			seller = u
			sellerCache[proj.SellerID] = u
		}
		_, plans, images, err = p.Projects.FindByID(ctx, proj.ID)
		if err != nil {
			return nil, 0, err
		}
		pcopy = proj
		out = append(out, ProjectView{Project: &pcopy, Plans: plans, Images: images, Seller: seller})
	}
	return out, total, nil
}
