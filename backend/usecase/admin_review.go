package usecase

import (
	"context"
	"strings"
	"time"

	"crowdfund-app/model"
	"crowdfund-app/repository"

	"github.com/google/uuid"
)

func (a *AdminUseCase) ListBuyerPending(ctx context.Context, limit, offset int) ([]UserView, int64, error) {
	var (
		rows  []model.User
		total int64
		out   []UserView
		err   error
		i     int
	)
	rows, total, err = a.Users.ListRoleStatus(ctx, model.UserRoleBuyer, model.AccountStatusPending, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out = make([]UserView, len(rows))
	for i = range rows {
		out[i] = UserViewFrom(rows[i])
	}
	return out, total, nil
}

func (a *AdminUseCase) ApproveBuyer(ctx context.Context, adminID, userID uuid.UUID, reason *string) (UserView, error) {
	var (
		u   model.User
		err error
		now time.Time
	)

	if err = a.requireReviewAdmin(ctx, adminID); err != nil {
		return UserView{}, err
	}
	u, err = a.Users.GetID(ctx, userID)
	if err != nil {
		return UserView{}, mapRepoNotFound(err)
	}
	if u.Role != model.UserRoleBuyer || u.AccountStatus != model.AccountStatusPending {
		return UserView{}, ErrInvalidState
	}
	now = time.Now().UTC()
	u.AccountStatus = model.AccountStatusActive
	u.UpdatedAt = now
	if err = a.Users.Update(ctx, &u); err != nil {
		return UserView{}, err
	}
	WriteAudit(ctx, a.Audit, adminID, "buyer_registration.approve", "user", u.ID.String(), reason, nil)
	a.writeReviewDecision(ctx, adminID, model.ReviewBuyerRegistration, userID, model.ReviewApproved, reason)
	return UserViewFrom(u), nil
}

func (a *AdminUseCase) RejectBuyer(ctx context.Context, adminID, userID uuid.UUID, reason *string) (UserView, error) {
	var (
		u   model.User
		err error
		now time.Time
	)

	if err = requireReviewReason(reason); err != nil {
		return UserView{}, err
	}
	reason = normalizeReviewReason(reason)
	if err = a.requireReviewAdmin(ctx, adminID); err != nil {
		return UserView{}, err
	}
	u, err = a.Users.GetID(ctx, userID)
	if err != nil {
		return UserView{}, mapRepoNotFound(err)
	}
	if u.Role != model.UserRoleBuyer || u.AccountStatus != model.AccountStatusPending {
		return UserView{}, ErrInvalidState
	}
	now = time.Now().UTC()
	u.AccountStatus = model.AccountStatusSuspended
	u.UpdatedAt = now
	if err = a.Users.Update(ctx, &u); err != nil {
		return UserView{}, err
	}
	WriteAudit(ctx, a.Audit, adminID, "buyer_registration.reject", "user", u.ID.String(), reason, nil)
	a.writeReviewDecision(ctx, adminID, model.ReviewBuyerRegistration, userID, model.ReviewRejected, reason)
	return UserViewFrom(u), nil
}

func (a *AdminUseCase) ListSellerRegs(ctx context.Context, status string, limit, offset int) ([]SellerRegItemView, int64, error) {
	var (
		rows  []model.SellerReg
		total int64
		out   []SellerRegItemView
		err   error
		i     int
		st    model.RegStatus
	)

	st = model.RegStatus(status)
	if !st.Valid() {
		st = model.RegPending
	}
	rows, total, err = a.SellerRegs.List(ctx, st, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out = make([]SellerRegItemView, len(rows))
	for i = range rows {
		out[i] = SellerRegItemFrom(rows[i])
	}
	return out, total, nil
}

func (a *AdminUseCase) GetSellerReg(ctx context.Context, id uuid.UUID) (*SellerRegView, error) {
	var (
		reg model.SellerReg
		err error
		u   model.User
		now time.Time
		out *SellerRegView
		tok string
	)

	reg, err = a.SellerRegs.GetID(ctx, id)
	if err != nil {
		return nil, mapRepoNotFound(err)
	}
	u, err = a.Users.GetID(ctx, reg.UserID)
	if err != nil {
		return nil, mapRepoNotFound(err)
	}
	now = time.Now().UTC()
	out = &SellerRegView{Registration: SellerRegItemFrom(reg), User: UserViewFrom(u)}
	if a.KYCAccess == nil {
		return out, nil
	}
	if reg.HasIDFront() {
		tok = a.KYCAccess.SignToken(*reg.IDFrontKey, now)
		out.FrontAccessToken = &tok
	}
	if reg.HasIDBack() {
		tok = a.KYCAccess.SignToken(*reg.IDBackKey, now)
		out.BackAccessToken = &tok
	}
	return out, nil
}

func (a *AdminUseCase) ApproveSellerReg(ctx context.Context, adminID, regID uuid.UUID) (SellerRegItemView, error) {
	var (
		reg model.SellerReg
		err error
		u   model.User
		now time.Time
	)

	if err = a.requireReviewAdmin(ctx, adminID); err != nil {
		return SellerRegItemView{}, err
	}
	reg, err = a.SellerRegs.GetID(ctx, regID)
	if err != nil {
		return SellerRegItemView{}, mapRepoNotFound(err)
	}
	if !reg.CanApprove() {
		return SellerRegItemView{}, ErrInvalidState
	}
	u, err = a.Users.GetID(ctx, reg.UserID)
	if err != nil {
		return SellerRegItemView{}, mapRepoNotFound(err)
	}
	now = time.Now().UTC()
	err = repository.WithTx(ctx, a.DB, func(txCtx context.Context) error {
		var txErr error

		u.AccountStatus = model.AccountStatusActive
		u.SellerStatus = model.SellerStatusActive
		u.UpdatedAt = now
		if txErr = a.Users.Update(txCtx, &u); txErr != nil {
			return txErr
		}
		reg.Status = model.RegApproved
		reg.ReviewedAt = &now
		reg.ReviewedBy = &adminID
		reg.UpdatedAt = now
		return a.SellerRegs.Update(txCtx, &reg)
	})
	if err != nil {
		return SellerRegItemView{}, err
	}
	WriteAudit(ctx, a.Audit, adminID, "seller_registration.approve", "seller_registration", reg.ID.String(), nil, nil)
	a.writeReviewDecision(ctx, adminID, model.ReviewSellerRegistration, regID, model.ReviewApproved, nil)
	return SellerRegItemFrom(reg), nil
}

func (a *AdminUseCase) RejectSellerReg(ctx context.Context, adminID, regID uuid.UUID, reason string) (SellerRegItemView, error) {
	var (
		reg model.SellerReg
		err error
		now time.Time
	)

	reason = strings.TrimSpace(reason)
	if reason == "" {
		return SellerRegItemView{}, ErrBadRequest
	}
	if err = a.requireReviewAdmin(ctx, adminID); err != nil {
		return SellerRegItemView{}, err
	}
	reg, err = a.SellerRegs.GetID(ctx, regID)
	if err != nil {
		return SellerRegItemView{}, mapRepoNotFound(err)
	}
	if reg.Status != model.RegPending {
		return SellerRegItemView{}, ErrInvalidState
	}
	now = time.Now().UTC()
	reg.Status = model.RegRejected
	reg.RejectReason = &reason
	reg.ReviewedAt = &now
	reg.ReviewedBy = &adminID
	reg.UpdatedAt = now
	if err = a.SellerRegs.Update(ctx, &reg); err != nil {
		return SellerRegItemView{}, err
	}
	WriteAudit(ctx, a.Audit, adminID, "seller_registration.reject", "seller_registration", reg.ID.String(), &reason, nil)
	a.writeReviewDecision(ctx, adminID, model.ReviewSellerRegistration, regID, model.ReviewRejected, &reason)
	return SellerRegItemFrom(reg), nil
}

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
		project   model.Project
		plans     []model.ProjectPlan
		images    []model.ProjectImage
		err       error
		now       time.Time
		action    string
		seller    model.User
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
	project, plans, images, err = a.Projects.GetID(ctx, projectID)
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
	if err = a.Projects.Update(ctx, &project); err != nil {
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
	seller, err = a.Users.GetID(ctx, project.SellerID)
	if err != nil {
		return nil, mapRepoNotFound(err)
	}
	return AssembleProjectViewPtr(project, plans, images, UserViewPtr(seller)), nil
}

func (a *AdminUseCase) DeleteProject(ctx context.Context, adminID, projectID uuid.UUID) error {
	var (
		project model.Project
		err     error
	)

	if err = a.requireReviewAdmin(ctx, adminID); err != nil {
		return err
	}
	if a.Projects == nil {
		return ErrNotFound
	}
	project, _, _, err = a.Projects.GetID(ctx, projectID)
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
		project model.Project
		plans   []model.ProjectPlan
		images  []model.ProjectImage
		err     error
		now     time.Time
		seller  model.User
	)

	if err = a.requireReviewAdmin(ctx, adminID); err != nil {
		return nil, err
	}
	if a.Projects == nil {
		return nil, ErrNotFound
	}
	project, plans, images, err = a.Projects.GetID(ctx, projectID)
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
		if err = a.Projects.Update(ctx, &project); err != nil {
			return nil, err
		}
		WriteAudit(ctx, a.Audit, adminID, "project.delete_request.approve", "project", project.ID.String(), reason, nil)
		a.writeReviewDecision(ctx, adminID, model.ReviewProjectDelete, projectID, model.ReviewApproved, reason)
	case "reject":
		project.Status = model.ProjectStatusActive
		project.DeleteRequested = false
		project.UpdatedAt = now
		if err = a.Projects.Update(ctx, &project); err != nil {
			return nil, err
		}
		WriteAudit(ctx, a.Audit, adminID, "project.delete_request.reject", "project", project.ID.String(), reason, nil)
		a.writeReviewDecision(ctx, adminID, model.ReviewProjectDelete, projectID, model.ReviewRejected, reason)
	default:
		return nil, ErrBadRequest
	}
	seller, err = a.Users.GetID(ctx, project.SellerID)
	if err != nil {
		return nil, mapRepoNotFound(err)
	}
	return AssembleProjectViewPtr(project, plans, images, UserViewPtr(seller)), nil
}
