package repository

import (
	"context"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const MaxSellerSupporterExport = 10_000

type SellerDashboardStats struct {
	TotalProjects   int64
	ActiveProjects  int64
	TotalSupporters int64
	TotalRevenue    int64
}

type ProjectAnalysisRow struct {
	ProjectID      uuid.UUID `gorm:"column:project_id"`
	Title          string    `gorm:"column:title"`
	SupporterCount int       `gorm:"column:supporter_count"`
	Revenue        int64     `gorm:"column:revenue"`
}

type PlanBreakdownRow struct {
	ProjectID      uuid.UUID `gorm:"column:project_id"`
	MembershipType string    `gorm:"column:membership_type"`
	Count          int       `gorm:"column:count"`
}

type SellerRepo interface {
	DashboardStats(ctx context.Context, sellerID uuid.UUID) (SellerDashboardStats, error)
	ListSupporters(ctx context.Context, sellerID uuid.UUID, projectID *uuid.UUID, limit, offset int) ([]entity.ProjectSupporter, int64, error)
	ListProjectAnalysis(ctx context.Context, sellerID uuid.UUID) ([]ProjectAnalysisRow, error)
	PlanBreakdownByProject(ctx context.Context, sellerID uuid.UUID) ([]PlanBreakdownRow, error)
}

type sellerRepo struct {
	db *gorm.DB
}

func NewSellerRepo(db *gorm.DB) SellerRepo {
	return &sellerRepo{db: db}
}

func (r *sellerRepo) dbCtx(ctx context.Context) *gorm.DB {
	return dbFromCtx(ctx, r.db)
}

func (r *sellerRepo) DashboardStats(ctx context.Context, sellerID uuid.UUID) (SellerDashboardStats, error) {
	var (
		out SellerDashboardStats
		err error
	)
	err = r.dbCtx(ctx).Model(&entity.Project{}).
		Where("seller_id = ?", sellerID).
		Count(&out.TotalProjects).Error
	if err != nil {
		return SellerDashboardStats{}, err
	}
	err = r.dbCtx(ctx).Model(&entity.Project{}).
		Where("seller_id = ? AND status = ?", sellerID, entity.ProjectStatusActive).
		Count(&out.ActiveProjects).Error
	if err != nil {
		return SellerDashboardStats{}, err
	}
	err = r.dbCtx(ctx).Model(&entity.Project{}).
		Where("seller_id = ?", sellerID).
		Select("COALESCE(SUM(supporter_count), 0)").
		Scan(&out.TotalSupporters).Error
	if err != nil {
		return SellerDashboardStats{}, err
	}
	err = r.dbCtx(ctx).Model(&entity.Purchase{}).
		Select("COALESCE(SUM(purchases.total_price), 0)").
		Joins("JOIN projects ON projects.id = purchases.project_id").
		Where("projects.seller_id = ?", sellerID).
		Where("purchases.status = ?", entity.PurchaseStatusPaid).
		Scan(&out.TotalRevenue).Error
	return out, err
}

func (r *sellerRepo) ListSupporters(ctx context.Context, sellerID uuid.UUID, projectID *uuid.UUID, limit, offset int) ([]entity.ProjectSupporter, int64, error) {
	var (
		q     *gorm.DB
		total int64
		rows  []entity.ProjectSupporter
		err   error
	)
	q = r.dbCtx(ctx).Model(&entity.ProjectSupporter{}).
		Joins("JOIN projects ON projects.id = project_supporters.project_id").
		Where("projects.seller_id = ?", sellerID)
	if projectID != nil {
		q = q.Where("project_supporters.project_id = ?", *projectID)
	}
	err = q.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > MaxSellerSupporterExport {
		limit = MaxSellerSupporterExport
	}
	if offset < 0 {
		offset = 0
	}
	err = q.Order("project_supporters.created_at DESC").
		Limit(limit).Offset(offset).
		Find(&rows).Error
	return rows, total, err
}

func (r *sellerRepo) ListProjectAnalysis(ctx context.Context, sellerID uuid.UUID) ([]ProjectAnalysisRow, error) {
	var (
		rows []ProjectAnalysisRow
		err  error
	)
	err = r.dbCtx(ctx).Model(&entity.Project{}).
		Select(`
			projects.id AS project_id,
			projects.title AS title,
			projects.supporter_count AS supporter_count,
			COALESCE(SUM(CASE WHEN purchases.status = ? THEN purchases.total_price ELSE 0 END), 0) AS revenue`,
			entity.PurchaseStatusPaid).
		Joins("LEFT JOIN purchases ON purchases.project_id = projects.id").
		Where("projects.seller_id = ?", sellerID).
		Group("projects.id, projects.title, projects.supporter_count").
		Order("projects.created_at DESC").
		Scan(&rows).Error
	return rows, err
}

func (r *sellerRepo) PlanBreakdownByProject(ctx context.Context, sellerID uuid.UUID) ([]PlanBreakdownRow, error) {
	var (
		rows []PlanBreakdownRow
		now  time.Time
		err  error
	)
	now = time.Now().UTC()
	err = r.dbCtx(ctx).Model(&entity.ProjectSupporter{}).
		Select("project_supporters.project_id, project_plans.membership_type, COUNT(*) AS count").
		Joins("JOIN projects ON projects.id = project_supporters.project_id").
		Joins("JOIN project_plans ON project_plans.id = project_supporters.plan_id").
		Where("projects.seller_id = ?", sellerID).
		Where("project_supporters.status IN ?", []entity.SupporterStatus{
			entity.SupporterStatusActive,
			entity.SupporterStatusPendingRenewal,
			entity.SupporterStatusCancelled,
		}).
		Where("project_supporters.current_period_end >= ?", now).
		Group("project_supporters.project_id, project_plans.membership_type").
		Scan(&rows).Error
	return rows, err
}
