package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"crowdfund-app/model"
	"crowdfund-app/repository"
	"crowdfund-app/validator"

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
	EnrichPurchase(ctx context.Context, row model.Purchase) (PurchaseView, error)
}

func (a *AdminUseCase) DashboardStats(ctx context.Context) (AdminStatsView, error) {
	var (
		stats  repository.AdminDashboardStats
		recent []model.User
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
		out.RecentUsers = append(out.RecentUsers, UserViewFrom(recent[i]))
	}
	return out, nil
}

func (a *AdminUseCase) ListUsers(ctx context.Context, role, search string, limit, offset int) ([]UserView, int64, error) {
	var (
		norm  validator.AdminUserListNormalized
		rows  []model.User
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
		out[i] = UserViewFrom(rows[i])
	}
	return out, total, nil
}

func (a *AdminUseCase) UpdateUser(ctx context.Context, adminID, userID uuid.UUID, in AdminUpdateUserIn) (UserView, error) {
	var (
		norm validator.AdminUpdateUserNormalized
		u    model.User
		err  error
	)
	norm, err = validator.ValidateAdminUpdateUser(validator.AdminUpdateUserInput{
		Email: in.Email, Name: in.Name, Role: in.Role, IsActive: in.IsActive,
	})
	if err != nil {
		return UserView{}, mapAdminCatalogErr(err)
	}
	u, err = a.Users.GetID(ctx, userID)
	if err != nil {
		return UserView{}, mapRepoNotFound(err)
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
		if u.Role == model.UserRoleSeller && u.SellerStatus == model.SellerStatusNotApplicable {
			u.SellerStatus = model.SellerStatusPending
		}
		if u.Role != model.UserRoleSeller {
			u.SellerStatus = model.SellerStatusNotApplicable
		}
		if u.Role != model.UserRoleAdmin {
			u.AdminRole = nil
		}
	}
	if norm.IsActiveSet && norm.IsActive != nil {
		if *norm.IsActive {
			u.AccountStatus = model.AccountStatusActive
		} else {
			u.AccountStatus = model.AccountStatusSuspended
		}
	}
	u.UpdatedAt = time.Now().UTC()
	if err = a.Users.Update(ctx, &u); err != nil {
		return UserView{}, err
	}
	WriteAudit(ctx, a.Audit, adminID, "user.update", "user", u.ID.String(), nil, nil)
	return UserViewFrom(u), nil
}

func (a *AdminUseCase) DeleteUser(ctx context.Context, adminID, userID uuid.UUID) error {
	var (
		u   model.User
		err error
	)
	u, err = a.Users.GetID(ctx, userID)
	if err != nil {
		return mapRepoNotFound(err)
	}
	if u.Role == model.UserRoleAdmin {
		return ErrForbidden
	}
	u.AccountStatus = model.AccountStatusBanned
	u.UpdatedAt = time.Now().UTC()
	if err = a.Users.Update(ctx, &u); err != nil {
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
		rows  []model.Purchase
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
		v, err = a.PurchaseViews.EnrichPurchase(ctx, rows[i])
		if err != nil {
			return nil, 0, err
		}
		out = append(out, v)
	}
	return out, total, nil
}

func (a *AdminUseCase) PatchPurchaseStatus(ctx context.Context, adminID, purchaseID uuid.UUID, status string) (PurchaseView, error) {
	var (
		to       model.PurchaseStatus
		purchase model.Purchase
		from     model.PurchaseStatus
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
	purchase, err = a.Purchases.GetID(ctx, purchaseID)
	if err != nil {
		return PurchaseView{}, mapRepoNotFound(err)
	}
	if purchase.Status == to {
		return a.PurchaseViews.EnrichPurchase(ctx, purchase)
	}
	from = purchase.Status
	purchase.Status = to
	purchase.UpdatedAt = time.Now().UTC()
	if err = a.Purchases.Update(ctx, &purchase); err != nil {
		return PurchaseView{}, err
	}
	if err = a.Purchases.RecordEvent(ctx, &model.PurchaseEvent{
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
	case errors.Is(err, model.ErrBadRole),
		errors.Is(err, model.ErrBadPurchase),
		errors.Is(err, model.ErrBadAccount),
		errors.Is(err, model.ErrBadEmail):
		return ErrBadRequest
	default:
		return err
	}
}

func strPtr(s string) *string {
	return &s
}
