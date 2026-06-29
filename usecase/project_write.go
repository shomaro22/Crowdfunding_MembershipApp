package usecase

import (
	"context"
	"strings"
	"time"

	"crowdfund-app/model"
	"crowdfund-app/repository"
	"crowdfund-app/validator"

	"github.com/google/uuid"
)

func (p *ProjectUseCase) Create(ctx context.Context, sellerID uuid.UUID, in CreateProjectIn) (*ProjectView, error) {
	var (
		u          *model.User
		err        error
		planInputs []model.PlanInput
		now        time.Time
		project       *model.Project
		plans      []model.ProjectPlan
		images     []model.ProjectImage
		i          int
		plan       PlanIn
	)

	u, err = p.Users.FindID(ctx, sellerID)
	if err != nil {
		return nil, mapRepoNotFound(err)
	}
	if u.Role != model.UserRoleSeller || u.SellerStatus != model.SellerStatusActive {
		return nil, ErrForbidden
	}
	planInputs = make([]model.PlanInput, len(in.Plans))
	for i, plan = range in.Plans {
		planInputs[i] = model.PlanInput{
			MembershipType: model.MembershipType(plan.MembershipType),
			Price:          plan.Price,
			MaxSupporters:  plan.MaxSupporters,
			Benefits:       plan.Benefits,
		}
	}
	if err = validator.ValidateProject(in.Title, in.Description, in.TargetAmount, model.DurationType(in.DurationType), in.StartDate, in.EndDate, planInputs); err != nil {
		return nil, mapProjectEntityErr(err)
	}
	now = time.Now().UTC()
	project = &model.Project{
		ID:           uuid.New(),
		SellerID:     sellerID,
		Title:        strings.TrimSpace(in.Title),
		Description:  strings.TrimSpace(in.Description),
		TargetAmount: in.TargetAmount,
		DurationType: model.DurationType(in.DurationType),
		StartDate:    in.StartDate.UTC(),
		EndDate:      in.EndDate,
		Status:       model.ProjectStatusPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	plans = make([]model.ProjectPlan, 0, len(in.Plans))
	for _, plan = range in.Plans {
		plans = append(plans, model.ProjectPlan{
			ID:             uuid.New(),
			MembershipType: model.MembershipType(plan.MembershipType),
			Price:          plan.Price,
			MaxSupporters:  plan.MaxSupporters,
			Benefits:       strings.TrimSpace(plan.Benefits),
			IsActive:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		})
	}
	images, project.MainImageURL, err = buildProjectImages(in.Images, in.ImageURL, now)
	if err != nil {
		return nil, mapProjectEntityErr(err)
	}
	if err = p.Projects.Create(ctx, project, plans, images); err != nil {
		return nil, err
	}
	return AssembleProjectViewPtr(project, plans, images, UserViewPtr(u)), nil
}

func (p *ProjectUseCase) Update(ctx context.Context, sellerID, projectID uuid.UUID, in UpdateProjectIn) (*ProjectView, error) {
	var (
		project   *model.Project
		plans  []model.ProjectPlan
		images []model.ProjectImage
		seller *model.User
		err    error
		now    time.Time
		dur    model.DurationType
		start  time.Time
		end    *time.Time
	)

	project, plans, images, err = p.Projects.FindID(ctx, projectID)
	if err != nil {
		return nil, mapRepoNotFound(err)
	}
	if project.SellerID != sellerID {
		return nil, ErrForbidden
	}
	now = time.Now().UTC()
	if in.Title != nil {
		if err = validator.ValidateTitle(*in.Title); err != nil {
			return nil, mapProjectEntityErr(err)
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
		dur = model.DurationType(*in.DurationType)
	}
	if in.StartDate != nil {
		start = in.StartDate.UTC()
	}
	if in.EndDate != nil {
		end = in.EndDate
	}
	if in.DurationType != nil || in.StartDate != nil || in.EndDate != nil {
		if err = validator.ValidateDuration(dur, start, end); err != nil {
			return nil, mapProjectEntityErr(err)
		}
		project.DurationType = dur
		project.StartDate = start
		project.EndDate = end
	}
	if in.Images != nil {
		images, project.MainImageURL, err = buildProjectImages(*in.Images, nil, now)
		if err != nil {
			return nil, mapProjectEntityErr(err)
		}
	} else if in.ImageURL != nil {
		images, project.MainImageURL, err = buildProjectImages(nil, in.ImageURL, now)
		if err != nil {
			return nil, mapProjectEntityErr(err)
		}
	}
	if in.IsReapply && project.Status == model.ProjectStatusActive {
		project.Status = model.ProjectStatusPending
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
	return AssembleProjectViewPtr(project, plans, images, UserViewPtr(seller)), nil
}

func (p *ProjectUseCase) RequestDelete(ctx context.Context, sellerID, projectID uuid.UUID) (*ProjectView, error) {
	var (
		project   *model.Project
		plans  []model.ProjectPlan
		images []model.ProjectImage
		seller *model.User
		err    error
		now    time.Time
	)

	project, plans, images, err = p.Projects.FindID(ctx, projectID)
	if err != nil {
		return nil, mapRepoNotFound(err)
	}
	if project.SellerID != sellerID {
		return nil, ErrForbidden
	}
	if project.Status != model.ProjectStatusActive {
		return nil, ErrInvalidState
	}
	now = time.Now().UTC()
	project.DeleteRequested = true
	project.Status = model.ProjectStatusDeleteRequested
	project.UpdatedAt = now
	if err = p.Projects.Update(ctx, project); err != nil {
		return nil, err
	}
	seller, _ = p.Users.FindID(ctx, project.SellerID)
	return AssembleProjectViewPtr(project, plans, images, UserViewPtr(seller)), nil
}
