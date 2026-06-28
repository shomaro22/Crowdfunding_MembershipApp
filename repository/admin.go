package repository

import (
	"context"

	"crowdfund-app/entity"

	"gorm.io/gorm"
)

const adminRecentUsersLimit = 5

type AdminDashboardStats struct {
	TotalUsers     int64
	TotalProjects  int64
	TotalPurchases int64
	TotalRevenue   int64
}

type AdminRepo interface {
	DashboardStats(ctx context.Context) (AdminDashboardStats, error)
	RecentUsers(ctx context.Context, limit int) ([]entity.User, error)
}

type adminRepo struct {
	db *gorm.DB
}

func NewAdminRepo(db *gorm.DB) AdminRepo {
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
	err = r.dbCtx(ctx).Model(&entity.User{}).Count(&out.TotalUsers).Error
	if err != nil {
		return AdminDashboardStats{}, err
	}
	err = r.dbCtx(ctx).Model(&entity.Project{}).Count(&out.TotalProjects).Error
	if err != nil {
		return AdminDashboardStats{}, err
	}
	err = r.dbCtx(ctx).Model(&entity.Purchase{}).Count(&out.TotalPurchases).Error
	if err != nil {
		return AdminDashboardStats{}, err
	}
	err = r.dbCtx(ctx).Model(&entity.Purchase{}).
		Where("status = ?", entity.PurchaseStatusPaid).
		Select("COALESCE(SUM(total_price), 0)").
		Scan(&out.TotalRevenue).Error
	return out, err
}

func (r *adminRepo) RecentUsers(ctx context.Context, limit int) ([]entity.User, error) {
	var (
		rows []entity.User
		err  error
	)
	if limit <= 0 {
		limit = adminRecentUsersLimit
	}
	err = r.dbCtx(ctx).Order("created_at DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

type AdminUserListFilter struct {
	Role   *entity.UserRole
	Search string
	Limit  int
	Offset int
}
