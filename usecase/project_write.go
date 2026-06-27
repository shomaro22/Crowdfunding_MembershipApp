package usecase

import (
	"context"
	"strings"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"

	"github.com/google/uuid"
)

func (p *ProjectUseCase) Create(ctx context.Context, sellerID uuid.UUID, in CreateProjectIn) (*ProjectView, error) {
	var (
		u          *entity.User
		err        error
		planInputs []entity.PlanInput
		now        time.Time
		project       *entity.Project
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
	project = &entity.Project{
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
	images, project.MainImageURL, err = buildProjectImages(in.Images, in.ImageURL, now)
	if err != nil {
		return nil, mapEntityErr(err)
	}
	if err = p.Projects.Create(ctx, project, plans, images); err != nil {
		return nil, err
	}
	return &ProjectView{Project: project, Plans: plans, Images: images, Seller: UserViewPtr(u)}, nil
}

func (p *ProjectUseCase) Update(ctx context.Context, sellerID, projectID uuid.UUID, in UpdateProjectIn) (*ProjectView, error) {
	var (
		project   *entity.Project
		plans  []entity.ProjectPlan
		images []entity.ProjectImage
		seller *entity.User
		err    error
		now    time.Time
		dur    entity.DurationType
		start  time.Time
		end    *time.Time
	)

	project, plans, images, err = p.Projects.FindID(ctx, projectID)
	if err != nil || project == nil {
		return nil, ErrNotFound
	}
	if project.SellerID != sellerID {
		return nil, ErrForbidden
	}
	now = time.Now().UTC()
	if in.Title != nil {
		if err = entity.ValidateTitle(*in.Title); err != nil {
			return nil, mapEntityErr(err)
		}
		project.Title = strings.TrimSpace(*in.Title)
	}
	if in.Description != nil {
		if strings.TrimSpace(*in.Description) == "" {
			return nil, ErrBadRequest
		}
		project.Description = strings.TrimSpace(*in.Description)
	}
	if in.TargetAmount != nil {
		if *in.TargetAmount < 0 {
			return nil, ErrBadRequest
		}
		project.TargetAmount = *in.TargetAmount
	}
	dur = project.DurationType
	start = project.StartDate
	end = project.EndDate
	if in.DurationType != nil {
		dur = *in.DurationType
	}
	if in.StartDate != nil {
		start = in.StartDate.UTC()
	}
	if in.EndDate != nil {
		end = in.EndDate
	}
	if in.DurationType != nil || in.StartDate != nil || in.EndDate != nil {
		if err = entity.ValidateDuration(dur, start, end); err != nil {
			return nil, mapEntityErr(err)
		}
		project.DurationType = dur
		project.StartDate = start
		project.EndDate = end
	}
	if in.Images != nil {
		images, project.MainImageURL, err = buildProjectImages(*in.Images, nil, now)
		if err != nil {
			return nil, mapEntityErr(err)
		}
	} else if in.ImageURL != nil {
		images, project.MainImageURL, err = buildProjectImages(nil, in.ImageURL, now)
		if err != nil {
			return nil, mapEntityErr(err)
		}
	}
	if in.IsReapply && project.Status == entity.ProjectStatusActive {
		project.Status = entity.ProjectStatusPending
	}
	project.UpdatedAt = now
	err = repository.WithTx(ctx, p.DB, func(txCtx context.Context) error {
		if err = p.Projects.Update(txCtx, project); err != nil {
			return err
		}
		if in.Images != nil || in.ImageURL != nil {
			return p.Projects.ReplaceImages(txCtx, projectID, images)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	seller, _ = p.Users.FindID(ctx, project.SellerID)
	return &ProjectView{Project: project, Plans: plans, Images: images, Seller: UserViewPtr(seller)}, nil
}

func (p *ProjectUseCase) RequestDelete(ctx context.Context, sellerID, projectID uuid.UUID) (*ProjectView, error) {
	var (
		project   *entity.Project
		plans  []entity.ProjectPlan
		images []entity.ProjectImage
		seller *entity.User
		err    error
		now    time.Time
	)

	project, plans, images, err = p.Projects.FindID(ctx, projectID)
	if err != nil || project == nil {
		return nil, ErrNotFound
	}
	if project.SellerID != sellerID {
		return nil, ErrForbidden
	}
	if project.Status != entity.ProjectStatusActive {
		return nil, ErrInvalidState
	}
	now = time.Now().UTC()
	project.DeleteRequested = true
	project.Status = entity.ProjectStatusDeleteRequested
	project.UpdatedAt = now
	if err = p.Projects.Update(ctx, project); err != nil {
		return nil, err
	}
	seller, _ = p.Users.FindID(ctx, project.SellerID)
	return &ProjectView{Project: project, Plans: plans, Images: images, Seller: UserViewPtr(seller)}, nil
}
