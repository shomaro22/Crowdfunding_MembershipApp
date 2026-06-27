package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdminUseCase struct {
	Users           repository.UserRepo
	SellerRegs      repository.SellerRegRepo
	Projects        repository.ProjectRepo
	Audit           repository.AuditRepo
	ReviewDecisions repository.ReviewDecisionRepo
	RefreshTokens   repository.RefreshTokenRepo
	TOTPBackup      repository.TOTPBackupRepo
	KYC             repository.KYCStore
	DB              *gorm.DB
	APIBase         string
	SignSecret      []byte
}

type SellerRegView struct {
	Registration *entity.SellerReg
	User         UserView
	FrontURL     *string
	BackURL      *string
}

func (a *AdminUseCase) requireReviewAdmin(ctx context.Context, adminID uuid.UUID) error {
	var (
		admin *entity.User
		err   error
	)

	admin, err = a.Users.FindID(ctx, adminID)
	if err != nil || admin == nil || admin.Role != entity.UserRoleAdmin {
		return ErrForbidden
	}
	if admin.AdminRole == nil {
		return ErrForbidden
	}
	switch *admin.AdminRole {
	case entity.AdminRoleSuperAdmin, entity.AdminRoleReviewAdmin:
		return nil
	default:
		return ErrForbidden
	}
}

func requireReviewReason(reason *string) error {
	if reason == nil || strings.TrimSpace(*reason) == "" {
		return ErrBadRequest
	}
	return nil
}

func normalizeReviewReason(reason *string) *string {
	var s string
	if reason == nil {
		return nil
	}
	s = strings.TrimSpace(*reason)
	if s == "" {
		return nil
	}
	return &s
}

func (a *AdminUseCase) ResetUser2FA(ctx context.Context, adminID, targetUserID uuid.UUID, reason *string) (UserView, error) {
	var (
		u   *entity.User
		err error
		now time.Time
	)

	u, err = a.Users.FindID(ctx, targetUserID)
	if err != nil || u == nil {
		return UserView{}, ErrNotFound
	}
	if !u.TOTPEnabled {
		return UserView{}, ErrInvalidState
	}
	now = time.Now().UTC()
	u.TOTPEnabled = false
	u.TOTPSecret = nil
	u.UpdatedAt = now
	if err = a.Users.Update(ctx, u); err != nil {
		return UserView{}, err
	}
	if a.TOTPBackup != nil {
		if err = a.TOTPBackup.DeleteUser(ctx, targetUserID); err != nil {
			return UserView{}, err
		}
	}
	if err = a.invalidateSessions(ctx, targetUserID); err != nil {
		return UserView{}, err
	}
	a.writeAudit(ctx, adminID, "user.reset_2fa", "user", u.ID.String(), reason, nil)
	return UserViewFrom(u), nil
}

func (a *AdminUseCase) ChangeAdminRole(ctx context.Context, actorID, targetUserID uuid.UUID, newRole entity.AdminRole, reason *string) (UserView, error) {
	var (
		actor   *entity.User
		u       *entity.User
		err     error
		now     time.Time
		oldRole string
		details string
	)

	if !newRole.Valid() {
		return UserView{}, ErrBadRequest
	}
	actor, err = a.Users.FindID(ctx, actorID)
	if err != nil || actor == nil || actor.Role != entity.UserRoleAdmin ||
		actor.AdminRole == nil || *actor.AdminRole != entity.AdminRoleSuperAdmin {
		return UserView{}, ErrForbidden
	}
	u, err = a.Users.FindID(ctx, targetUserID)
	if err != nil || u == nil {
		return UserView{}, ErrNotFound
	}
	if u.Role != entity.UserRoleAdmin {
		return UserView{}, ErrInvalidState
	}
	if u.AdminRole != nil && *u.AdminRole == newRole {
		return UserViewFrom(u), nil
	}
	if u.AdminRole != nil {
		oldRole = string(*u.AdminRole)
	}
	now = time.Now().UTC()
	role := newRole
	u.AdminRole = &role
	u.UpdatedAt = now
	if err = a.Users.Update(ctx, u); err != nil {
		return UserView{}, err
	}
	if err = a.invalidateSessions(ctx, targetUserID); err != nil {
		return UserView{}, err
	}
	details = fmt.Sprintf("%s -> %s", oldRole, newRole)
	a.writeAudit(ctx, actorID, "user.change_admin_role", "user", u.ID.String(), reason, &details)
	return UserViewFrom(u), nil
}

func (a *AdminUseCase) invalidateSessions(ctx context.Context, userID uuid.UUID) error {
	var err error

	return repository.WithTx(ctx, a.DB, func(txCtx context.Context) error {
		if _, err = a.Users.BumpTokenVer(txCtx, userID); err != nil {
			return err
		}
		if a.RefreshTokens == nil {
			return nil
		}
		return a.RefreshTokens.RevokeAllByUser(txCtx, userID, time.Now().UTC())
	})
}

func (a *AdminUseCase) writeReviewDecision(ctx context.Context, reviewerID uuid.UUID, resourceType string, resourceID uuid.UUID, decision string, reason *string) {
	if a.ReviewDecisions == nil {
		return
	}
	_ = a.ReviewDecisions.Create(ctx, &entity.ReviewDecision{
		ID:           uuid.New(),
		ReviewerID:   reviewerID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Decision:     decision,
		Reason:       reason,
		CreatedAt:    time.Now().UTC(),
	})
}

func (a *AdminUseCase) writeAudit(ctx context.Context, adminID uuid.UUID, action, resource, resourceID string, reason, details *string) {
	if a.Audit == nil {
		return
	}
	_ = a.Audit.Create(ctx, &entity.AuditLog{
		ID:         uuid.New(),
		AdminID:    adminID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Reason:     reason,
		Details:    details,
		CreatedAt:  time.Now().UTC(),
	})
}
