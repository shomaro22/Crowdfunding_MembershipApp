package usecase

import (
	"context"
	"strings"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
)

func (p *ProjectUseCase) Create(ctx context.Context, sellerID uuid.UUID, in CreateProjectIn) (*ProjectView, error) {
	var (
		u          *entity.User
		err        error
		planInputs []entity.PlanInput
		now        time.Time
		proj       *entity.Project
		plans      []entity.ProjectPlan
		images     []entity.ProjectImage
		i          int
		pl         PlanIn
	)

	u, err = p.Users.FindID(ctx, sellerID)
	if err != nil || u == nil {
		return nil, ErrNotFound
	}
	if u.Role != entity.UserRoleSeller || u.SellerStatus != entity.SellerStatusActive {
		return nil, ErrForbidden
	}
	planInputs = make([]entity.PlanInput, len(in.Plans))
	for i, pl = range in.Plans {
		planInputs[i] = entity.PlanInput{
			MembershipType: pl.MembershipType,
			Price:          pl.Price,
			MaxSupporters:  pl.MaxSupporters,
			Benefits:       pl.Benefits,
		}
	}
	if err = entity.ValidateProject(in.Title, in.Description, in.TargetAmount, in.DurationType, in.StartDate, in.EndDate, planInputs); err != nil {
		return nil, mapEntityErr(err)
	}
	now = time.Now().UTC()
	proj = &entity.Project{
		ID:           uuid.New(),
		SellerID:     sellerID,
		Title:        strings.TrimSpace(in.Title),
		Description:  strings.TrimSpace(in.Description),
		TargetAmount: in.TargetAmount,
		DurationType: in.DurationType,
		StartDate:    in.StartDate.UTC(),
		EndDate:      in.EndDate,
		Status:       entity.ProjectStatusPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	plans = make([]entity.ProjectPlan, 0, len(in.Plans))
	for _, pl = range in.Plans {
		plans = append(plans, entity.ProjectPlan{
			ID:             uuid.New(),
			MembershipType: pl.MembershipType,
			Price:          pl.Price,
			MaxSupporters:  pl.MaxSupporters,
			Benefits:       strings.TrimSpace(pl.Benefits),
			IsActive:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		})
	}
	images, proj.MainImageURL, err = buildProjectImages(in.Images, in.ImageURL, now)
	if err != nil {
		return nil, mapEntityErr(err)
	}
	if err = p.Projects.Create(ctx, proj, plans, images); err != nil {
		return nil, err
	}
	return &ProjectView{Project: proj, Plans: plans, Images: images, Seller: u}, nil
}
