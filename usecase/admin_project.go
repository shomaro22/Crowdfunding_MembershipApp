package usecase

import (
	"context"
	"strings"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
)

func (a *AdminUseCase) ListProjects(ctx context.Context, status *entity.ProjectStatus, limit, offset int) ([]ProjectView, int64, error) {
	var (
		rows  []entity.Project
		total int64
		err   error
		uc    *ProjectUseCase
	)

	if a.Projects == nil {
		return nil, 0, ErrNotFound
	}
	rows, total, err = a.Projects.ListAll(ctx, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	uc = &ProjectUseCase{Projects: a.Projects, Users: a.Users}
	return uc.enrichList(ctx, rows, total)
}

func (a *AdminUseCase) PatchProject(ctx context.Context, adminID, projectID uuid.UUID, status entity.ProjectStatus, reason *string) (*ProjectView, error) {
	var (
		proj   *entity.Project
		plans  []entity.ProjectPlan
		images []entity.ProjectImage
		err    error
		now    time.Time
		action string
		seller *entity.User
	)

	if err = a.requireReviewAdmin(ctx, adminID); err != nil {
		return nil, err
	}
	if a.Projects == nil {
		return nil, ErrNotFound
	}
	proj, plans, images, err = a.Projects.FindByID(ctx, projectID)
	if err != nil || proj == nil {
		return nil, ErrNotFound
	}
	if !status.Valid() {
		return nil, ErrBadRequest
	}
	switch status {
	case entity.ProjectStatusActive:
		if proj.Status != entity.ProjectStatusPending {
			return nil, ErrInvalidState
		}
	case entity.ProjectStatusCancelled:
		if proj.Status != entity.ProjectStatusPending {
			return nil, ErrInvalidState
		}
	default:
		return nil, ErrInvalidState
	}
	now = time.Now().UTC()
	proj.Status = status
	proj.UpdatedAt = now
	if err = a.Projects.Update(ctx, proj); err != nil {
		return nil, err
	}
	action = "project.approve"
	if status == entity.ProjectStatusCancelled {
		action = "project.reject"
	}
	a.writeAudit(ctx, adminID, action, "project", proj.ID.String(), reason, nil)
	if status == entity.ProjectStatusActive {
		a.writeReviewDecision(ctx, adminID, entity.ReviewProject, projectID, entity.ReviewApproved, reason)
	} else {
		a.writeReviewDecision(ctx, adminID, entity.ReviewProject, projectID, entity.ReviewRejected, reason)
	}
	seller, _ = a.Users.FindByID(ctx, proj.SellerID)
	return &ProjectView{Project: proj, Plans: plans, Images: images, Seller: seller}, nil
}

func (a *AdminUseCase) DeleteProject(ctx context.Context, adminID, projectID uuid.UUID) error {
	var (
		proj *entity.Project
		err  error
	)

	if err = a.requireReviewAdmin(ctx, adminID); err != nil {
		return err
	}
	if a.Projects == nil {
		return ErrNotFound
	}
	proj, _, _, err = a.Projects.FindByID(ctx, projectID)
	if err != nil || proj == nil {
		return ErrNotFound
	}
	if err = a.Projects.Delete(ctx, projectID); err != nil {
		return err
	}
	a.writeAudit(ctx, adminID, "project.delete", "project", projectID.String(), nil, nil)
	return nil
}

func (a *AdminUseCase) PatchDeleteReq(ctx context.Context, adminID, projectID uuid.UUID, action string, reason *string) (*ProjectView, error) {
	var (
		proj   *entity.Project
		plans  []entity.ProjectPlan
		images []entity.ProjectImage
		err    error
		now    time.Time
		seller *entity.User
	)

	if err = a.requireReviewAdmin(ctx, adminID); err != nil {
		return nil, err
	}
	if a.Projects == nil {
		return nil, ErrNotFound
	}
	proj, plans, images, err = a.Projects.FindByID(ctx, projectID)
	if err != nil || proj == nil {
		return nil, ErrNotFound
	}
	if proj.Status != entity.ProjectStatusDeleteRequested {
		return nil, ErrInvalidState
	}
	now = time.Now().UTC()
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "approve":
		proj.Status = entity.ProjectStatusDeleted
		proj.DeleteRequested = false
		proj.UpdatedAt = now
		if err = a.Projects.Update(ctx, proj); err != nil {
			return nil, err
		}
		a.writeAudit(ctx, adminID, "project.delete_request.approve", "project", proj.ID.String(), reason, nil)
		a.writeReviewDecision(ctx, adminID, entity.ReviewProjectDelete, projectID, entity.ReviewApproved, reason)
	case "reject":
		proj.Status = entity.ProjectStatusActive
		proj.DeleteRequested = false
		proj.UpdatedAt = now
		if err = a.Projects.Update(ctx, proj); err != nil {
			return nil, err
		}
		a.writeAudit(ctx, adminID, "project.delete_request.reject", "project", proj.ID.String(), reason, nil)
		a.writeReviewDecision(ctx, adminID, entity.ReviewProjectDelete, projectID, entity.ReviewRejected, reason)
	default:
		return nil, ErrBadRequest
	}
	seller, _ = a.Users.FindByID(ctx, proj.SellerID)
	return &ProjectView{Project: proj, Plans: plans, Images: images, Seller: seller}, nil
}
