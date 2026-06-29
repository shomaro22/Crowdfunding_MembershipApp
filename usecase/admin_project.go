package usecase

import (
	"context"
	"strings"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
)

func (a *AdminUseCase) ListProjects(ctx context.Context, status *string, limit, offset int) ([]ProjectView, int64, error) {
	var (
		rows   []entity.Project
		total  int64
		err    error
		filter *entity.ProjectStatus
		st     entity.ProjectStatus
	)

	if a.Projects == nil || a.ProjectsUC == nil {
		return nil, 0, ErrNotFound
	}
	if status != nil && *status != "" {
		st = entity.ProjectStatus(*status)
		if !st.Valid() {
			return nil, 0, ErrBadRequest
		}
		filter = &st
	}
	rows, total, err = a.Projects.ListAll(ctx, filter, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return a.ProjectsUC.EnrichList(ctx, rows, total)
}

func (a *AdminUseCase) PatchProject(ctx context.Context, adminID, projectID uuid.UUID, status string, reason *string) (*ProjectView, error) {
	var (
		project   *entity.Project
		plans     []entity.ProjectPlan
		images    []entity.ProjectImage
		err       error
		now       time.Time
		action    string
		seller    *entity.User
		projectSt entity.ProjectStatus
	)

	projectSt = entity.ProjectStatus(status)
	if !projectSt.Valid() {
		return nil, ErrBadRequest
	}
	if err = a.requireReviewAdmin(ctx, adminID); err != nil {
		return nil, err
	}
	if a.Projects == nil {
		return nil, ErrNotFound
	}
	project, plans, images, err = a.Projects.FindID(ctx, projectID)
	if err != nil {
		return nil, mapRepoNotFound(err)
	}
	switch projectSt {
	case entity.ProjectStatusActive:
		if project.Status != entity.ProjectStatusPending {
			return nil, ErrInvalidState
		}
	case entity.ProjectStatusCancelled:
		if project.Status != entity.ProjectStatusPending {
			return nil, ErrInvalidState
		}
		if err = requireReviewReason(reason); err != nil {
			return nil, err
		}
		reason = normalizeReviewReason(reason)
	default:
		return nil, ErrInvalidState
	}
	now = time.Now().UTC()
	project.Status = projectSt
	project.UpdatedAt = now
	if err = a.Projects.Update(ctx, project); err != nil {
		return nil, err
	}
	action = "project.approve"
	if projectSt == entity.ProjectStatusCancelled {
		action = "project.reject"
	}
	WriteAudit(ctx, a.Audit, adminID, action, "project", project.ID.String(), reason, nil)
	if projectSt == entity.ProjectStatusActive {
		a.writeReviewDecision(ctx, adminID, entity.ReviewProject, projectID, entity.ReviewApproved, reason)
	} else {
		a.writeReviewDecision(ctx, adminID, entity.ReviewProject, projectID, entity.ReviewRejected, reason)
	}
	seller, _ = a.Users.FindID(ctx, project.SellerID)
	return AssembleProjectViewPtr(project, plans, images, UserViewPtr(seller)), nil
}

func (a *AdminUseCase) DeleteProject(ctx context.Context, adminID, projectID uuid.UUID) error {
	var (
		project *entity.Project
		err  error
	)

	if err = a.requireReviewAdmin(ctx, adminID); err != nil {
		return err
	}
	if a.Projects == nil {
		return ErrNotFound
	}
	project, _, _, err = a.Projects.FindID(ctx, projectID)
	if err != nil {
		return mapRepoNotFound(err)
	}
	switch project.Status {
	case entity.ProjectStatusPending, entity.ProjectStatusCancelled, entity.ProjectStatusDeleted:
	default:
		return ErrInvalidState
	}
	if err = a.Projects.Delete(ctx, projectID); err != nil {
		return err
	}
	WriteAudit(ctx, a.Audit, adminID, "project.delete", "project", projectID.String(), nil, nil)
	return nil
}

func (a *AdminUseCase) PatchDeleteReq(ctx context.Context, adminID, projectID uuid.UUID, action string, reason *string) (*ProjectView, error) {
	var (
		project   *entity.Project
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
	project, plans, images, err = a.Projects.FindID(ctx, projectID)
	if err != nil {
		return nil, mapRepoNotFound(err)
	}
	if project.Status != entity.ProjectStatusDeleteRequested {
		return nil, ErrInvalidState
	}
	now = time.Now().UTC()
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "approve":
		project.Status = entity.ProjectStatusDeleted
		project.DeleteRequested = false
		project.UpdatedAt = now
		if err = a.Projects.Update(ctx, project); err != nil {
			return nil, err
		}
		WriteAudit(ctx, a.Audit, adminID, "project.delete_request.approve", "project", project.ID.String(), reason, nil)
		a.writeReviewDecision(ctx, adminID, entity.ReviewProjectDelete, projectID, entity.ReviewApproved, reason)
	case "reject":
		project.Status = entity.ProjectStatusActive
		project.DeleteRequested = false
		project.UpdatedAt = now
		if err = a.Projects.Update(ctx, project); err != nil {
			return nil, err
		}
		WriteAudit(ctx, a.Audit, adminID, "project.delete_request.reject", "project", project.ID.String(), reason, nil)
		a.writeReviewDecision(ctx, adminID, entity.ReviewProjectDelete, projectID, entity.ReviewRejected, reason)
	default:
		return nil, ErrBadRequest
	}
	seller, _ = a.Users.FindID(ctx, project.SellerID)
	return AssembleProjectViewPtr(project, plans, images, UserViewPtr(seller)), nil
}
