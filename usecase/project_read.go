package usecase

import (
	"context"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
)

func (p *ProjectUseCase) FindView(ctx context.Context, id uuid.UUID, viewerID *uuid.UUID, viewerRole entity.UserRole) (*ProjectView, error) {
	var (
		proj   *entity.Project
		plans  []entity.ProjectPlan
		images []entity.ProjectImage
		seller *entity.User
		err    error
		now    = time.Now().UTC()
	)

	proj, plans, images, err = p.Projects.FindID(ctx, id)
	if err != nil || proj == nil {
		return nil, ErrNotFound
	}
	if !canViewProject(proj, viewerID, viewerRole, now) {
		return nil, ErrNotFound
	}
	seller, err = p.Users.FindID(ctx, proj.SellerID)
	if err != nil {
		return nil, err
	}
	return &ProjectView{Project: proj, Plans: plans, Images: images, Seller: seller}, nil
}

func (p *ProjectUseCase) ListPlans(ctx context.Context, projectID uuid.UUID, viewerID *uuid.UUID, viewerRole entity.UserRole) ([]entity.ProjectPlan, error) {
	var (
		proj  *entity.Project
		plans []entity.ProjectPlan
		err   error
		now   time.Time
	)

	proj, plans, _, err = p.Projects.FindID(ctx, projectID)
	if err != nil || proj == nil {
		return nil, ErrNotFound
	}
	now = time.Now().UTC()
	if !canViewProject(proj, viewerID, viewerRole, now) {
		return nil, ErrNotFound
	}
	return plans, nil
}
