package usecase

import (
	"context"

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
