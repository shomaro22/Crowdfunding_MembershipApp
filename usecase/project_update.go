package usecase

import (
	"context"
	"strings"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
)

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

	proj, plans, images, err = p.Projects.FindByID(ctx, projectID)
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
	if err = p.Projects.Update(ctx, proj); err != nil {
		return nil, err
	}
	if in.Images != nil || in.ImageURL != nil {
		if err = p.Projects.ReplaceImages(ctx, projectID, images); err != nil {
			return nil, err
		}
	}
	seller, _ = p.Users.FindByID(ctx, proj.SellerID)
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

	proj, plans, images, err = p.Projects.FindByID(ctx, projectID)
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
	seller, _ = p.Users.FindByID(ctx, proj.SellerID)
	return &ProjectView{Project: proj, Plans: plans, Images: images, Seller: seller}, nil
}
