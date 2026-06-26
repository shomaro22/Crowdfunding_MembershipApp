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

func (p *ProjectUseCase) Update(ctx context.Context, sellerID, projectID uuid.UUID, in UpdateProjectIn) (*ProjectView, error) {
	var (
		proj   *entity.Project
		plans  []entity.ProjectPlan
		images []entity.ProjectImage
		seller *entity.User
		err    error
		now    time.Time
		dur    entity.DurationType
		start  time.Time
		end    *time.Time
	)

	proj, plans, images, err = p.Projects.FindID(ctx, projectID)
	if err != nil || proj == nil {
		return nil, ErrNotFound
	}
	if proj.SellerID != sellerID {
		return nil, ErrForbidden
	}
	now = time.Now().UTC()
	if in.Title != nil {
		if err = entity.ValidateTitle(*in.Title); err != nil {
			return nil, mapEntityErr(err)
		}
		proj.Title = strings.TrimSpace(*in.Title)
	}
	if in.Description != nil {
		if strings.TrimSpace(*in.Description) == "" {
			return nil, ErrBadRequest
		}
		proj.Description = strings.TrimSpace(*in.Description)
	}
	if in.TargetAmount != nil {
		if *in.TargetAmount < 0 {
			return nil, ErrBadRequest
		}
		proj.TargetAmount = *in.TargetAmount
	}
	dur = proj.DurationType
	start = proj.StartDate
	end = proj.EndDate
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
		proj.DurationType = dur
		proj.StartDate = start
		proj.EndDate = end
	}
	if in.Images != nil {
		images, proj.MainImageURL, err = buildProjectImages(*in.Images, nil, now)
		if err != nil {
			return nil, mapEntityErr(err)
		}
	} else if in.ImageURL != nil {
		images, proj.MainImageURL, err = buildProjectImages(nil, in.ImageURL, now)
		if err != nil {
			return nil, mapEntityErr(err)
		}
	}
	if in.IsReapply && proj.Status == entity.ProjectStatusActive {
		proj.Status = entity.ProjectStatusPending
	}
	proj.UpdatedAt = now
	err = repository.WithTx(ctx, p.DB, func(txCtx context.Context) error {
		if err = p.Projects.Update(txCtx, proj); err != nil {
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
	seller, _ = p.Users.FindID(ctx, proj.SellerID)
	return &ProjectView{Project: proj, Plans: plans, Images: images, Seller: seller}, nil
}

func (p *ProjectUseCase) RequestDelete(ctx context.Context, sellerID, projectID uuid.UUID) (*ProjectView, error) {
	var (
		proj   *entity.Project
		plans  []entity.ProjectPlan
		images []entity.ProjectImage
		seller *entity.User
		err    error
		now    time.Time
	)

	proj, plans, images, err = p.Projects.FindID(ctx, projectID)
	if err != nil || proj == nil {
		return nil, ErrNotFound
	}
	if proj.SellerID != sellerID {
		return nil, ErrForbidden
	}
	if proj.Status != entity.ProjectStatusActive {
		return nil, ErrInvalidState
	}
	now = time.Now().UTC()
	proj.DeleteRequested = true
	proj.Status = entity.ProjectStatusDeleteRequested
	proj.UpdatedAt = now
	if err = p.Projects.Update(ctx, proj); err != nil {
		return nil, err
	}
	seller, _ = p.Users.FindID(ctx, proj.SellerID)
	return &ProjectView{Project: proj, Plans: plans, Images: images, Seller: seller}, nil
}
