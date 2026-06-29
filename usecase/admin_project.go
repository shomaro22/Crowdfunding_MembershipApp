package usecase

import (
	"context"
	"strings"
	"time"

	"crowdfund-app/model"

	"github.com/google/uuid"
)

func (a *AdminUseCase) ListProjects(ctx context.Context, status *string, limit, offset int) ([]ProjectView, int64, error) {
	var (
		rows   []model.Project
		total  int64
		err    error
		filter *model.ProjectStatus
		st     model.ProjectStatus
	)

	if a.Projects == nil || a.ProjectsUC == nil {
		return nil, 0, ErrNotFound
	}
	if status != nil && *status != "" {
		st = model.ProjectStatus(*status)
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
		project   *model.Project
		plans     []model.ProjectPlan
		images    []model.ProjectImage
		err       error
		now       time.Time
		action    string
		seller    *model.User
		projectSt model.ProjectStatus
	)

	projectSt = model.ProjectStatus(status)
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
	case model.ProjectStatusActive:
		if project.Status != model.ProjectStatusPending {
			return nil, ErrInvalidState
		}
	case model.ProjectStatusCancelled:
		if project.Status != model.ProjectStatusPending {
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
	if projectSt == model.ProjectStatusCancelled {
		action = "project.reject"
	}
	WriteAudit(ctx, a.Audit, adminID, action, "project", project.ID.String(), reason, nil)
	if projectSt == model.ProjectStatusActive {
		a.writeReviewDecision(ctx, adminID, model.ReviewProject, projectID, model.ReviewApproved, reason)
	} else {
		a.writeReviewDecision(ctx, adminID, model.ReviewProject, projectID, model.ReviewRejected, reason)
	}
	seller, _ = a.Users.FindID(ctx, project.SellerID)
	return AssembleProjectViewPtr(project, plans, images, UserViewPtr(seller)), nil
}

func (a *AdminUseCase) DeleteProject(ctx context.Context, adminID, projectID uuid.UUID) error {
	var (
		project *model.Project
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
	case model.ProjectStatusPending, model.ProjectStatusCancelled, model.ProjectStatusDeleted:
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
		project   *model.Project
		plans  []model.ProjectPlan
		images []model.ProjectImage
		err    error
		now    time.Time
		seller *model.User
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
	if project.Status != model.ProjectStatusDeleteRequested {
		return nil, ErrInvalidState
	}
	now = time.Now().UTC()
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "approve":
		project.Status = model.ProjectStatusDeleted
		project.DeleteRequested = false
		project.UpdatedAt = now
		if err = a.Projects.Update(ctx, project); err != nil {
			return nil, err
		}
		WriteAudit(ctx, a.Audit, adminID, "project.delete_request.approve", "project", project.ID.String(), reason, nil)
		a.writeReviewDecision(ctx, adminID, model.ReviewProjectDelete, projectID, model.ReviewApproved, reason)
	case "reject":
		project.Status = model.ProjectStatusActive
		project.DeleteRequested = false
		project.UpdatedAt = now
		if err = a.Projects.Update(ctx, project); err != nil {
			return nil, err
		}
		WriteAudit(ctx, a.Audit, adminID, "project.delete_request.reject", "project", project.ID.String(), reason, nil)
		a.writeReviewDecision(ctx, adminID, model.ReviewProjectDelete, projectID, model.ReviewRejected, reason)
	default:
		return nil, ErrBadRequest
	}
	seller, _ = a.Users.FindID(ctx, project.SellerID)
	return AssembleProjectViewPtr(project, plans, images, UserViewPtr(seller)), nil
}
