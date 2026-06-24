package usecase

import (
	"context"
	"strings"

	"crowdfund-app/entity"

	"github.com/google/uuid"
)

func (a *AdminUseCase) requireReviewAdmin(ctx context.Context, adminID uuid.UUID) error {
	var (
		admin *entity.User
		err   error
	)

	admin, err = a.Users.FindByID(ctx, adminID)
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
