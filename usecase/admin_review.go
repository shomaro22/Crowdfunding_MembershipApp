package usecase

import (
	"context"
	"strings"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"

	"github.com/google/uuid"
)

func (a *AdminUseCase) ListBuyerPending(ctx context.Context, limit, offset int) ([]UserView, int64, error) {
	var (
		rows  []entity.User
		total int64
		out   []UserView
		err   error
		i     int
	)
	rows, total, err = a.Users.ListRoleStatus(ctx, entity.UserRoleBuyer, entity.AccountStatusPending, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out = make([]UserView, len(rows))
	for i = range rows {
		out[i] = UserViewFrom(&rows[i])
	}
	return out, total, nil
}

func (a *AdminUseCase) ApproveBuyer(ctx context.Context, adminID, userID uuid.UUID, reason *string) (UserView, error) {
	var (
		u   *entity.User
		err error
		now time.Time
	)

	if err = a.requireReviewAdmin(ctx, adminID); err != nil {
		return UserView{}, err
	}
	u, err = a.Users.FindID(ctx, userID)
	if err != nil || u == nil {
		return UserView{}, ErrNotFound
	}
	if u.Role != entity.UserRoleBuyer || u.AccountStatus != entity.AccountStatusPending {
		return UserView{}, ErrInvalidState
	}
	now = time.Now().UTC()
	u.AccountStatus = entity.AccountStatusActive
	u.UpdatedAt = now
	if err = a.Users.Update(ctx, u); err != nil {
		return UserView{}, err
	}
	a.writeAudit(ctx, adminID, "buyer_registration.approve", "user", u.ID.String(), reason, nil)
	a.writeReviewDecision(ctx, adminID, entity.ReviewBuyerRegistration, userID, entity.ReviewApproved, reason)
	return UserViewFrom(u), nil
}

func (a *AdminUseCase) RejectBuyer(ctx context.Context, adminID, userID uuid.UUID, reason *string) (UserView, error) {
	var (
		u   *entity.User
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
	u, err = a.Users.FindID(ctx, userID)
	if err != nil || u == nil {
		return UserView{}, ErrNotFound
	}
	if u.Role != entity.UserRoleBuyer || u.AccountStatus != entity.AccountStatusPending {
		return UserView{}, ErrInvalidState
	}
	now = time.Now().UTC()
	u.AccountStatus = entity.AccountStatusSuspended
	u.UpdatedAt = now
	if err = a.Users.Update(ctx, u); err != nil {
		return UserView{}, err
	}
	a.writeAudit(ctx, adminID, "buyer_registration.reject", "user", u.ID.String(), reason, nil)
	a.writeReviewDecision(ctx, adminID, entity.ReviewBuyerRegistration, userID, entity.ReviewRejected, reason)
	return UserViewFrom(u), nil
}

func (a *AdminUseCase) ListSellerRegs(ctx context.Context, status string, limit, offset int) ([]SellerRegItemView, int64, error) {
	var (
		rows  []entity.SellerReg
		total int64
		out   []SellerRegItemView
		err   error
		i     int
		st    entity.RegStatus
	)

	st = entity.RegStatus(status)
	if !st.Valid() {
		st = entity.RegPending
	}
	rows, total, err = a.SellerRegs.List(ctx, st, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out = make([]SellerRegItemView, len(rows))
	for i = range rows {
		out[i] = SellerRegItemFrom(&rows[i])
	}
	return out, total, nil
}

func (a *AdminUseCase) GetSellerReg(ctx context.Context, id uuid.UUID) (*SellerRegView, error) {
	var (
		reg *entity.SellerReg
		err error
		u   *entity.User
		now time.Time
		out *SellerRegView
		tok string
	)

	reg, err = a.SellerRegs.FindID(ctx, id)
	if err != nil || reg == nil {
		return nil, ErrNotFound
	}
	u, err = a.Users.FindID(ctx, reg.UserID)
	if err != nil || u == nil {
		return nil, ErrNotFound
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
		reg *entity.SellerReg
		err error
		u   *entity.User
		now time.Time
	)

	if err = a.requireReviewAdmin(ctx, adminID); err != nil {
		return SellerRegItemView{}, err
	}
	reg, err = a.SellerRegs.FindID(ctx, regID)
	if err != nil || reg == nil {
		return SellerRegItemView{}, ErrNotFound
	}
	if !reg.CanApprove() {
		return SellerRegItemView{}, ErrInvalidState
	}
	u, err = a.Users.FindID(ctx, reg.UserID)
	if err != nil || u == nil {
		return SellerRegItemView{}, ErrNotFound
	}
	now = time.Now().UTC()
	err = repository.WithTx(ctx, a.DB, func(txCtx context.Context) error {
		var txErr error

		u.AccountStatus = entity.AccountStatusActive
		u.SellerStatus = entity.SellerStatusActive
		u.UpdatedAt = now
		if txErr = a.Users.Update(txCtx, u); txErr != nil {
			return txErr
		}
		reg.Status = entity.RegApproved
		reg.ReviewedAt = &now
		reg.ReviewedBy = &adminID
		reg.UpdatedAt = now
		return a.SellerRegs.Update(txCtx, reg)
	})
	if err != nil {
		return SellerRegItemView{}, err
	}
	a.writeAudit(ctx, adminID, "seller_registration.approve", "seller_registration", reg.ID.String(), nil, nil)
	a.writeReviewDecision(ctx, adminID, entity.ReviewSellerRegistration, regID, entity.ReviewApproved, nil)
	return SellerRegItemFrom(reg), nil
}

func (a *AdminUseCase) RejectSellerReg(ctx context.Context, adminID, regID uuid.UUID, reason string) (SellerRegItemView, error) {
	var (
		reg *entity.SellerReg
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
	reg, err = a.SellerRegs.FindID(ctx, regID)
	if err != nil || reg == nil {
		return SellerRegItemView{}, ErrNotFound
	}
	if reg.Status != entity.RegPending {
		return SellerRegItemView{}, ErrInvalidState
	}
	now = time.Now().UTC()
	reg.Status = entity.RegRejected
	reg.RejectReason = &reason
	reg.ReviewedAt = &now
	reg.ReviewedBy = &adminID
	reg.UpdatedAt = now
	if err = a.SellerRegs.Update(ctx, reg); err != nil {
		return SellerRegItemView{}, err
	}
	a.writeAudit(ctx, adminID, "seller_registration.reject", "seller_registration", reg.ID.String(), &reason, nil)
	a.writeReviewDecision(ctx, adminID, entity.ReviewSellerRegistration, regID, entity.ReviewRejected, &reason)
	return SellerRegItemFrom(reg), nil
}
