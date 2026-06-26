package usecase

import (
	"context"
	"fmt"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"

	"github.com/google/uuid"
)

func (a *AdminUseCase) ResetUser2FA(ctx context.Context, adminID, targetUserID uuid.UUID, reason *string) (*entity.User, error) {
	var (
		u   *entity.User
		err error
		now time.Time
	)

	u, err = a.Users.FindID(ctx, targetUserID)
	if err != nil || u == nil {
		return nil, ErrNotFound
	}
	if !u.TOTPEnabled {
		return nil, ErrInvalidState
	}
	now = time.Now().UTC()
	u.TOTPEnabled = false
	u.TOTPSecret = nil
	u.UpdatedAt = now
	if err = a.Users.Update(ctx, u); err != nil {
		return nil, err
	}
	if a.TOTPBackup != nil {
		if err = a.TOTPBackup.DeleteUser(ctx, targetUserID); err != nil {
			return nil, err
		}
	}
	if err = a.invalidateSessions(ctx, targetUserID); err != nil {
		return nil, err
	}
	a.writeAudit(ctx, adminID, "user.reset_2fa", "user", u.ID.String(), reason, nil)
	return u, nil
}

func (a *AdminUseCase) ChangeAdminRole(ctx context.Context, actorID, targetUserID uuid.UUID, newRole entity.AdminRole, reason *string) (*entity.User, error) {
	var (
		actor   *entity.User
		u       *entity.User
		err     error
		now     time.Time
		oldRole string
		details string
	)

	if !newRole.Valid() {
		return nil, ErrBadRequest
	}
	actor, err = a.Users.FindID(ctx, actorID)
	if err != nil || actor == nil || actor.Role != entity.UserRoleAdmin ||
		actor.AdminRole == nil || *actor.AdminRole != entity.AdminRoleSuperAdmin {
		return nil, ErrForbidden
	}
	u, err = a.Users.FindID(ctx, targetUserID)
	if err != nil || u == nil {
		return nil, ErrNotFound
	}
	if u.Role != entity.UserRoleAdmin {
		return nil, ErrInvalidState
	}
	if u.AdminRole != nil && *u.AdminRole == newRole {
		return u, nil
	}
	if u.AdminRole != nil {
		oldRole = string(*u.AdminRole)
	}
	now = time.Now().UTC()
	role := newRole
	u.AdminRole = &role
	u.UpdatedAt = now
	if err = a.Users.Update(ctx, u); err != nil {
		return nil, err
	}
	if err = a.invalidateSessions(ctx, targetUserID); err != nil {
		return nil, err
	}
	details = fmt.Sprintf("%s -> %s", oldRole, newRole)
	a.writeAudit(ctx, actorID, "user.change_admin_role", "user", u.ID.String(), reason, &details)
	return u, nil
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
		return a.RefreshTokens.RevokeUser(txCtx, userID, time.Now().UTC())
	})
}
