package usecase

import (
	"context"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"

	"github.com/google/uuid"
)

func (p *ProjectUseCase) ListPublic(ctx context.Context, search, sort string, limit, offset int) ([]ProjectView, int64, error) {
	var (
		rows  []entity.Project
		total int64
		err   error
	)

	rows, total, err = p.Projects.ListPublic(ctx, repository.ProjectListFilter{
		Search: search,
		Sort:   sort,
		Limit:  limit,
		Offset: offset,
		AsOf:   time.Now().UTC(),
	})
	if err != nil {
		return nil, 0, err
	}
	return p.enrichList(ctx, rows, total)
}

func (p *ProjectUseCase) ListMine(ctx context.Context, sellerID uuid.UUID, limit, offset int) ([]ProjectView, int64, error) {
	var (
		u     *entity.User
		err   error
		rows  []entity.Project
		total int64
	)

	u, err = p.Users.FindID(ctx, sellerID)
	if err != nil || u == nil || u.Role != entity.UserRoleSeller {
		return nil, 0, ErrForbidden
	}
	rows, total, err = p.Projects.ListSeller(ctx, sellerID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return p.enrichList(ctx, rows, total)
}

func canViewProject(p *entity.Project, viewerID *uuid.UUID, role entity.UserRole, now time.Time) bool {
	if p.IsPublic(now) {
		return true
	}
	if viewerID != nil && *viewerID == p.SellerID {
		return true
	}
	if role == entity.UserRoleAdmin {
		return true
	}
	return false
}
