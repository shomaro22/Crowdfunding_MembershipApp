package usecase

import (
	"context"
	"strings"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"

	"github.com/google/uuid"
)

func (a *AdminUseCase) ListBuyerPending(ctx context.Context, limit, offset int) ([]entity.User, int64, error) {
	return a.Users.ListRoleStatus(ctx, entity.UserRoleBuyer, entity.AccountStatusPending, limit, offset)
}

func (a *AdminUseCase) ApproveBuyer(ctx context.Context, adminID, userID uuid.UUID, reason *string) (*entity.User, error) {
	var (
		u   *entity.User
		err error
		now time.Time
	)

	if err = a.requireReviewAdmin(ctx, adminID); err != nil {
		return nil, err
	}
	u, err = a.Users.FindID(ctx, userID)
	if err != nil || u == nil {
		return nil, ErrNotFound
	}
	if u.Role != entity.UserRoleBuyer || u.AccountStatus != entity.AccountStatusPending {
		return nil, ErrInvalidState
	}
	now = time.Now().UTC()
	u.AccountStatus = entity.AccountStatusActive
	u.UpdatedAt = now
	if err = a.Users.Update(ctx, u); err != nil {
		return nil, err
	}
	a.writeAudit(ctx, adminID, "buyer_registration.approve", "user", u.ID.String(), reason, nil)
	a.writeReviewDecision(ctx, adminID, entity.ReviewBuyerRegistration, userID, entity.ReviewApproved, reason)
	return u, nil
}

func (a *AdminUseCase) RejectBuyer(ctx context.Context, adminID, userID uuid.UUID, reason *string) (*entity.User, error) {
	var (
		u   *entity.User
		err error
		now time.Time
	)

	if err = requireReviewReason(reason); err != nil {
		return nil, err
	}
	reason = normalizeReviewReason(reason)
	if err = a.requireReviewAdmin(ctx, adminID); err != nil {
		return nil, err
	}
	u, err = a.Users.FindID(ctx, userID)
	if err != nil || u == nil {
		return nil, ErrNotFound
	}
	if u.Role != entity.UserRoleBuyer || u.AccountStatus != entity.AccountStatusPending {
		return nil, ErrInvalidState
	}
	now = time.Now().UTC()
	u.AccountStatus = entity.AccountStatusSuspended
	u.UpdatedAt = now
	if err = a.Users.Update(ctx, u); err != nil {
		return nil, err
	}
	a.writeAudit(ctx, adminID, "buyer_registration.reject", "user", u.ID.String(), reason, nil)
	a.writeReviewDecision(ctx, adminID, entity.ReviewBuyerRegistration, userID, entity.ReviewRejected, reason)
	return u, nil
}

func (a *AdminUseCase) ListSellerRegs(ctx context.Context, status entity.RegStatus, limit, offset int) ([]entity.SellerReg, int64, error) {
	if !status.Valid() {
		status = entity.RegPending
	}
	return a.SellerRegs.List(ctx, status, limit, offset)
}

func (a *AdminUseCase) GetSellerReg(ctx context.Context, id uuid.UUID) (*SellerRegView, error) {
	var (
		reg *entity.SellerReg
		err error
		u   *entity.User
		now time.Time
		out *SellerRegView
		tok string
		url string
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
	out = &SellerRegView{Registration: reg, User: u}
	if reg.HasIDFront() {
		tok = repository.SignKYCAccess(a.SignSecret, *reg.IDFrontKey, now)
		url = repository.KYCAccessURL(a.APIBase, tok)
		out.FrontURL = &url
	}
	if reg.HasIDBack() {
		tok = repository.SignKYCAccess(a.SignSecret, *reg.IDBackKey, now)
		url = repository.KYCAccessURL(a.APIBase, tok)
		out.BackURL = &url
	}
	return out, nil
}

func (a *AdminUseCase) ApproveSellerReg(ctx context.Context, adminID, regID uuid.UUID) (*entity.SellerReg, error) {
	var (
		reg *entity.SellerReg
		err error
		u   *entity.User
		now time.Time
	)

	if err = a.requireReviewAdmin(ctx, adminID); err != nil {
		return nil, err
	}
	reg, err = a.SellerRegs.FindID(ctx, regID)
	if err != nil || reg == nil {
		return nil, ErrNotFound
	}
	if !reg.CanApprove() {
		return nil, ErrInvalidState
	}
	u, err = a.Users.FindID(ctx, reg.UserID)
	if err != nil || u == nil {
		return nil, ErrNotFound
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
		return nil, err
	}
	a.writeAudit(ctx, adminID, "seller_registration.approve", "seller_registration", reg.ID.String(), nil, nil)
	a.writeReviewDecision(ctx, adminID, entity.ReviewSellerRegistration, regID, entity.ReviewApproved, nil)
	return reg, nil
}

func (a *AdminUseCase) RejectSellerReg(ctx context.Context, adminID, regID uuid.UUID, reason string) (*entity.SellerReg, error) {
	var (
		reg *entity.SellerReg
		err error
		now time.Time
	)

	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, ErrBadRequest
	}
	if err = a.requireReviewAdmin(ctx, adminID); err != nil {
		return nil, err
	}
	reg, err = a.SellerRegs.FindID(ctx, regID)
	if err != nil || reg == nil {
		return nil, ErrNotFound
	}
	if reg.Status != entity.RegPending {
		return nil, ErrInvalidState
	}
	now = time.Now().UTC()
	reg.Status = entity.RegRejected
	reg.RejectReason = &reason
	reg.ReviewedAt = &now
	reg.ReviewedBy = &adminID
	reg.UpdatedAt = now
	if err = a.SellerRegs.Update(ctx, reg); err != nil {
		return nil, err
	}
	a.writeAudit(ctx, adminID, "seller_registration.reject", "seller_registration", reg.ID.String(), &reason, nil)
	a.writeReviewDecision(ctx, adminID, entity.ReviewSellerRegistration, regID, entity.ReviewRejected, &reason)
	return reg, nil
}
