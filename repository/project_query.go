package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProjectListFilter struct {
	Search string
	Sort   string
	Limit  int
	Offset int
	AsOf   time.Time // 公開一覧の期限判定基準。zero なら ListPublic 内で UTC 現在時刻を使う。
}

func (r *projectRepo) FindID(ctx context.Context, id uuid.UUID) (*entity.Project, []entity.ProjectPlan, []entity.ProjectImage, error) {
	var (
		p      entity.Project
		plans  []entity.ProjectPlan
		images []entity.ProjectImage
		err    error
	)

	err = r.dbCtx(ctx).Where("id = ?", id).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, nil, nil
	}
	if err != nil {
		return nil, nil, nil, err
	}
	if err = r.dbCtx(ctx).Where("project_id = ?", id).Order("membership_type ASC").Find(&plans).Error; err != nil {
		return nil, nil, nil, err
	}
	if err = r.dbCtx(ctx).Where("project_id = ?", id).Order("sort_order ASC").Find(&images).Error; err != nil {
		return nil, nil, nil, err
	}
	return &p, plans, images, nil
}

func (r *projectRepo) ListPublic(ctx context.Context, f ProjectListFilter) ([]entity.Project, int64, error) {
	var (
		asOf   time.Time
		q      *gorm.DB
		total  int64
		limit  int
		offset int
		order  string
		rows   []entity.Project
		s      string
		like   string
		err    error
	)

	asOf = f.AsOf
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	q = r.dbCtx(ctx).Model(&entity.Project{}).
		Where("status = ?", entity.ProjectStatusActive).
		Where("(duration_type = ? OR (duration_type = ? AND end_date >= ?))",
			entity.DurationUnlimited, entity.DurationLimited, asOf)
	if s = strings.TrimSpace(f.Search); s != "" {
		like = "%" + s + "%"
		q = q.Where("(title ILIKE ? OR description ILIKE ?)", like, like)
	}
	if err = q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	limit, offset = paginate(f.Limit, f.Offset)
	order = publicSort(f.Sort)
	err = q.Order(order).Limit(limit).Offset(offset).Find(&rows).Error
	return rows, total, err
}

func publicSort(sort string) string {
	switch strings.ToLower(strings.TrimSpace(sort)) {
	case "supporters":
		return "supporter_count DESC, created_at DESC"
	case "ending_soon":
		return "end_date ASC NULLS LAST, created_at DESC"
	default:
		return "created_at DESC"
	}
}

func (r *projectRepo) ListSeller(ctx context.Context, sellerID uuid.UUID, limit, offset int) ([]entity.Project, int64, error) {
	var (
		q     *gorm.DB
		total int64
		rows  []entity.Project
		err   error
	)
	q = r.dbCtx(ctx).Model(&entity.Project{}).Where("seller_id = ?", sellerID)
	err = q.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	limit, offset = paginate(limit, offset)
	err = r.dbCtx(ctx).Where("seller_id = ?", sellerID).
		Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, total, err
}

func (r *projectRepo) ListAll(ctx context.Context, status *entity.ProjectStatus, limit, offset int) ([]entity.Project, int64, error) {
	var (
		q     *gorm.DB
		total int64
		rows  []entity.Project
		db    *gorm.DB
		err   error
	)
	q = r.dbCtx(ctx).Model(&entity.Project{})
	if status != nil && status.Valid() {
		q = q.Where("status = ?", *status)
	}
	err = q.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	limit, offset = paginate(limit, offset)
	db = r.dbCtx(ctx).Order("created_at DESC").Limit(limit).Offset(offset)
	if status != nil && status.Valid() {
		db = db.Where("status = ?", *status)
	}
	err = db.Find(&rows).Error
	return rows, total, err
}

func (r *projectRepo) FindPlan(ctx context.Context, projectID, planID uuid.UUID) (*entity.ProjectPlan, error) {
	var (
		plan entity.ProjectPlan
		err  error
	)
	err = r.dbCtx(ctx).Where("id = ? AND project_id = ?", planID, projectID).First(&plan).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

func paginate(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
