package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"crowdfund-app/model"
	"crowdfund-app/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdminUseCase struct {
	Users           repository.IUserRepo
	SellerRegs      repository.ISellerRegRepo
	Projects        repository.IProjectRepo
	Audit           repository.IAuditRepo
	ReviewDecisions repository.IReviewDecisionRepo
	RefreshTokens   repository.IRefreshTokenRepo
	TOTPBackup      repository.ITOTPBackupRepo
	KYC             repository.IKYCStore
	ProjectsUC      *ProjectUseCase
	KYCAccess       *KYCAccessUseCase
	AdminStats      repository.IAdminRepo
	Purchases       repository.IPurchaseRepo
	PurchaseViews   PurchaseEnricher
	DB              *gorm.DB
}

type SellerRegView struct {
	Registration     SellerRegItemView
	User             UserView
	FrontAccessToken *string
	BackAccessToken  *string
}

func (a *AdminUseCase) requireReviewAdmin(ctx context.Context, adminID uuid.UUID) error {
	var (
		admin model.User
		err   error
	)

	admin, err = a.Users.GetID(ctx, adminID)
	if err != nil {
		return mapRepoNotFoundAs(err, ErrForbidden)
	}
	if admin.Role != model.UserRoleAdmin {
		return ErrForbidden
	}
	if admin.AdminRole == nil {
		return ErrForbidden
	}
	switch *admin.AdminRole {
	case model.AdminRoleSuperAdmin, model.AdminRoleReviewAdmin:
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
		u   model.User
		err error
		now time.Time
	)

	u, err = a.Users.GetID(ctx, targetUserID)
	if err != nil {
		return UserView{}, mapRepoNotFound(err)
	}
	if !u.TOTPEnabled {
		return UserView{}, ErrInvalidState
	}
	now = time.Now().UTC()
	u.TOTPEnabled = false
	u.TOTPSecret = nil
	u.UpdatedAt = now
	if err = a.Users.Update(ctx, &u); err != nil {
		return UserView{}, err
	}
	if a.TOTPBackup != nil {
		if err = a.TOTPBackup.DeleteUser(ctx, targetUserID); err != nil {
			return UserView{}, err
		}
	}
	if err = InvalidateUserSessions(ctx, a.DB, a.Users, a.RefreshTokens, targetUserID); err != nil {
		return UserView{}, err
	}
	WriteAudit(ctx, a.Audit, adminID, "user.reset_2fa", "user", u.ID.String(), reason, nil)
	return UserViewFrom(u), nil
}

func (a *AdminUseCase) ChangeAdminRole(ctx context.Context, actorID, targetUserID uuid.UUID, newRole string, reason *string) (UserView, error) {
	var (
		actor   model.User
		u       model.User
		err     error
		now     time.Time
		oldRole string
		details string
		role    model.AdminRole
	)

	role = model.AdminRole(newRole)
	if !role.Valid() {
		return UserView{}, ErrBadRequest
	}
	actor, err = a.Users.GetID(ctx, actorID)
	if err != nil {
		return UserView{}, mapRepoNotFoundAs(err, ErrForbidden)
	}
	if actor.Role != model.UserRoleAdmin ||
		actor.AdminRole == nil || *actor.AdminRole != model.AdminRoleSuperAdmin {
		return UserView{}, ErrForbidden
	}
	u, err = a.Users.GetID(ctx, targetUserID)
	if err != nil {
		return UserView{}, mapRepoNotFound(err)
	}
	if u.Role != model.UserRoleAdmin {
		return UserView{}, ErrInvalidState
	}
	if u.AdminRole != nil && *u.AdminRole == role {
		return UserViewFrom(u), nil
	}
	if u.AdminRole != nil {
		oldRole = string(*u.AdminRole)
	}
	now = time.Now().UTC()
	roleCopy := role
	u.AdminRole = &roleCopy
	u.UpdatedAt = now
	if err = a.Users.Update(ctx, &u); err != nil {
		return UserView{}, err
	}
	if err = InvalidateUserSessions(ctx, a.DB, a.Users, a.RefreshTokens, targetUserID); err != nil {
		return UserView{}, err
	}
	details = fmt.Sprintf("%s -> %s", oldRole, role)
	WriteAudit(ctx, a.Audit, actorID, "user.change_admin_role", "user", u.ID.String(), reason, &details)
	return UserViewFrom(u), nil
}

func (a *AdminUseCase) writeReviewDecision(ctx context.Context, reviewerID uuid.UUID, resourceType string, resourceID uuid.UUID, decision string, reason *string) {
	if a.ReviewDecisions == nil {
		return
	}
	_ = a.ReviewDecisions.Create(ctx, &model.ReviewDecision{
		ID:           uuid.New(),
		ReviewerID:   reviewerID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Decision:     decision,
		Reason:       reason,
		CreatedAt:    time.Now().UTC(),
	})
}
