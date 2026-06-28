package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"
	"crowdfund-app/validator"

	"github.com/google/uuid"
)

type AdminStatsView struct {
	TotalUsers     int64
	TotalProjects  int64
	TotalPurchases int64
	TotalRevenue   int64
	RecentUsers    []UserView
}

type AdminUpdateUserIn struct {
	Email    *string
	Name     *string
	Role     *string
	IsActive *bool
}

type PurchaseEnricher interface {
	EnrichPurchase(ctx context.Context, row *entity.Purchase) (PurchaseView, error)
}

func (a *AdminUseCase) DashboardStats(ctx context.Context) (AdminStatsView, error) {
	var (
		stats  repository.AdminDashboardStats
		recent []entity.User
		out    AdminStatsView
		i      int
		err    error
	)
	if a == nil || a.AdminStats == nil {
		return AdminStatsView{}, ErrBadRequest
	}
	stats, err = a.AdminStats.DashboardStats(ctx)
	if err != nil {
		return AdminStatsView{}, err
	}
	recent, err = a.AdminStats.RecentUsers(ctx, 5)
	if err != nil {
		return AdminStatsView{}, err
	}
	out = AdminStatsView{
		TotalUsers:     stats.TotalUsers,
		TotalProjects:  stats.TotalProjects,
		TotalPurchases: stats.TotalPurchases,
		TotalRevenue:   stats.TotalRevenue,
		RecentUsers:    make([]UserView, 0, len(recent)),
	}
	for i = range recent {
		out.RecentUsers = append(out.RecentUsers, UserViewFrom(&recent[i]))
	}
	return out, nil
}

func (a *AdminUseCase) ListUsers(ctx context.Context, role, search string, limit, offset int) ([]UserView, int64, error) {
	var (
		norm  validator.AdminUserListNormalized
		rows  []entity.User
		total int64
		out   []UserView
		i     int
		err   error
	)
	norm, err = validator.ValidateAdminUserList(validator.AdminUserListInput{
		Role: role, Search: search, Limit: limit, Offset: offset,
	})
	if err != nil {
		return nil, 0, mapAdminCatalogErr(err)
	}
	rows, total, err = a.Users.ListAdmin(ctx, repository.AdminUserListFilter{
		Role: norm.Role, Search: norm.Search, Limit: norm.Limit, Offset: norm.Offset,
	})
	if err != nil {
		return nil, 0, err
	}
	out = make([]UserView, len(rows))
	for i = range rows {
		out[i] = UserViewFrom(&rows[i])
	}
	return out, total, nil
}

func (a *AdminUseCase) UpdateUser(ctx context.Context, adminID, userID uuid.UUID, in AdminUpdateUserIn) (UserView, error) {
	var (
		norm validator.AdminUpdateUserNormalized
		u    *entity.User
		err  error
	)
	norm, err = validator.ValidateAdminUpdateUser(validator.AdminUpdateUserInput{
		Email: in.Email, Name: in.Name, Role: in.Role, IsActive: in.IsActive,
	})
	if err != nil {
		return UserView{}, mapAdminCatalogErr(err)
	}
	u, err = a.Users.FindID(ctx, userID)
	if err != nil || u == nil {
		return UserView{}, ErrNotFound
	}
	if norm.Email != nil && *norm.Email != u.Email {
		var exists bool
		exists, err = a.Users.EmailExists(ctx, *norm.Email)
		if err != nil {
			return UserView{}, err
		}
		if exists {
			return UserView{}, ErrEmailInUse
		}
		u.Email = *norm.Email
	}
	if norm.NameSet {
		u.Name = norm.Name
	}
	if norm.RoleSet && norm.Role != nil {
		u.Role = *norm.Role
		if u.Role == entity.UserRoleSeller && u.SellerStatus == entity.SellerStatusNotApplicable {
			u.SellerStatus = entity.SellerStatusPending
		}
		if u.Role != entity.UserRoleSeller {
			u.SellerStatus = entity.SellerStatusNotApplicable
		}
		if u.Role != entity.UserRoleAdmin {
			u.AdminRole = nil
		}
	}
	if norm.IsActiveSet && norm.IsActive != nil {
		if *norm.IsActive {
			u.AccountStatus = entity.AccountStatusActive
		} else {
			u.AccountStatus = entity.AccountStatusSuspended
		}
	}
	u.UpdatedAt = time.Now().UTC()
	if err = a.Users.Update(ctx, u); err != nil {
		return UserView{}, err
	}
	WriteAudit(ctx, a.Audit, adminID, "user.update", "user", u.ID.String(), nil, nil)
	return UserViewFrom(u), nil
}

func (a *AdminUseCase) DeleteUser(ctx context.Context, adminID, userID uuid.UUID) error {
	var (
		u   *entity.User
		err error
	)
	u, err = a.Users.FindID(ctx, userID)
	if err != nil || u == nil {
		return ErrNotFound
	}
	if u.Role == entity.UserRoleAdmin {
		return ErrForbidden
	}
	u.AccountStatus = entity.AccountStatusBanned
	u.UpdatedAt = time.Now().UTC()
	if err = a.Users.Update(ctx, u); err != nil {
		return err
	}
	if err = InvalidateUserSessions(ctx, a.DB, a.Users, a.RefreshTokens, userID); err != nil {
		return err
	}
	WriteAudit(ctx, a.Audit, adminID, "user.delete", "user", u.ID.String(), nil, nil)
	return nil
}

func (a *AdminUseCase) ListPurchases(ctx context.Context, status string, limit, offset int) ([]PurchaseView, int64, error) {
	var (
		norm  validator.AdminPurchaseListNormalized
		rows  []entity.Purchase
		total int64
		out   []PurchaseView
		i     int
		v     PurchaseView
		err   error
	)
	if a == nil || a.Purchases == nil || a.PurchaseViews == nil {
		return nil, 0, ErrBadRequest
	}
	norm, err = validator.ValidateAdminPurchaseList(validator.AdminPurchaseListInput{
		Status: status, Limit: limit, Offset: offset,
	})
	if err != nil {
		return nil, 0, mapAdminCatalogErr(err)
	}
	rows, total, err = a.Purchases.ListAll(ctx, norm.Status, norm.Limit, norm.Offset)
	if err != nil {
		return nil, 0, err
	}
	out = make([]PurchaseView, 0, len(rows))
	for i = range rows {
		v, err = a.PurchaseViews.EnrichPurchase(ctx, &rows[i])
		if err != nil {
			return nil, 0, err
		}
		out = append(out, v)
	}
	return out, total, nil
}

func (a *AdminUseCase) PatchPurchaseStatus(ctx context.Context, adminID, purchaseID uuid.UUID, status string) (PurchaseView, error) {
	var (
		to       entity.PurchaseStatus
		purchase *entity.Purchase
		from     entity.PurchaseStatus
		details  string
		view     PurchaseView
		err      error
	)
	if a == nil || a.Purchases == nil || a.PurchaseViews == nil {
		return PurchaseView{}, ErrBadRequest
	}
	to, err = validator.ValidateAdminPurchaseStatus(status)
	if err != nil {
		return PurchaseView{}, mapAdminCatalogErr(err)
	}
	purchase, err = a.Purchases.FindID(ctx, purchaseID)
	if err != nil || purchase == nil {
		return PurchaseView{}, ErrNotFound
	}
	if purchase.Status == to {
		return a.PurchaseViews.EnrichPurchase(ctx, purchase)
	}
	from = purchase.Status
	purchase.Status = to
	purchase.UpdatedAt = time.Now().UTC()
	if err = a.Purchases.Update(ctx, purchase); err != nil {
		return PurchaseView{}, err
	}
	if err = a.Purchases.RecordEvent(ctx, &entity.PurchaseEvent{
		ID:         uuid.New(),
		PurchaseID: purchase.ID,
		FromStatus: &from,
		ToStatus:   to,
		Reason:     strPtr("admin update"),
		CreatedAt:  time.Now().UTC(),
	}); err != nil {
		return PurchaseView{}, err
	}
	details = fmt.Sprintf("%s -> %s", from, to)
	WriteAudit(ctx, a.Audit, adminID, "purchase.update_status", "purchase", purchase.ID.String(), nil, &details)
	view, err = a.PurchaseViews.EnrichPurchase(ctx, purchase)
	return view, err
}

func mapAdminCatalogErr(err error) error {
	switch {
	case errors.Is(err, entity.ErrBadRole),
		errors.Is(err, entity.ErrBadPurchase),
		errors.Is(err, entity.ErrBadAccount),
		errors.Is(err, entity.ErrBadEmail):
		return ErrBadRequest
	default:
		return err
	}
}

func strPtr(s string) *string {
	return &s
}
