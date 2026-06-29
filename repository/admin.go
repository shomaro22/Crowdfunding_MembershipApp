package repository

import (
	"context"

	"crowdfund-app/model"

	"gorm.io/gorm"
)

const adminRecentUsersLimit = 5

type AdminDashboardStats struct {
	TotalUsers     int64
	TotalProjects  int64
	TotalPurchases int64
	TotalRevenue   int64
}

type IAdminRepo interface {
	DashboardStats(ctx context.Context) (AdminDashboardStats, error)
	RecentUsers(ctx context.Context, limit int) ([]model.User, error)
}

type adminRepo struct {
	db *gorm.DB
}

func NewAdminRepo(db *gorm.DB) IAdminRepo {
	return &adminRepo{db: db}
}

func (r *adminRepo) dbCtx(ctx context.Context) *gorm.DB {
	return dbFromCtx(ctx, r.db)
}

func (r *adminRepo) DashboardStats(ctx context.Context) (AdminDashboardStats, error) {
	var (
		out AdminDashboardStats
		err error
	)
	err = r.dbCtx(ctx).Model(&model.User{}).Count(&out.TotalUsers).Error
	if err != nil {
		return AdminDashboardStats{}, err
	}
	err = r.dbCtx(ctx).Model(&model.Project{}).Count(&out.TotalProjects).Error
	if err != nil {
		return AdminDashboardStats{}, err
	}
	err = r.dbCtx(ctx).Model(&model.Purchase{}).Count(&out.TotalPurchases).Error
	if err != nil {
		return AdminDashboardStats{}, err
	}
	err = r.dbCtx(ctx).Model(&model.Purchase{}).
		Where("status = ?", model.PurchaseStatusPaid).
		Select("COALESCE(SUM(total_price), 0)").
		Scan(&out.TotalRevenue).Error
	return out, err
}

func (r *adminRepo) RecentUsers(ctx context.Context, limit int) ([]model.User, error) {
	var (
		rows []model.User
		err  error
	)
	if limit <= 0 {
		limit = adminRecentUsersLimit
	}
	err = r.dbCtx(ctx).Order("created_at DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

type AdminUserListFilter struct {
	Role   *model.UserRole
	Search string
	Limit  int
	Offset int
}
