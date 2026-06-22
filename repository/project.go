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
	Now    time.Time
}

type ProjectRepo interface {
	Create(ctx context.Context, p *entity.Project, plans []entity.ProjectPlan, images []entity.ProjectImage) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Project, []entity.ProjectPlan, []entity.ProjectImage, error)
	ListPublic(ctx context.Context, f ProjectListFilter) ([]entity.Project, int64, error)
	ListBySeller(ctx context.Context, sellerID uuid.UUID, limit, offset int) ([]entity.Project, int64, error)
	ListAll(ctx context.Context, status *entity.ProjectStatus, limit, offset int) ([]entity.Project, int64, error)
	Update(ctx context.Context, p *entity.Project) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type projectRepo struct {
	db *gorm.DB
}

func NewProjectRepo(db *gorm.DB) ProjectRepo {
	return &projectRepo{db: db}
}

func (r *projectRepo) dbCtx(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok && tx != nil {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *projectRepo) Create(ctx context.Context, p *entity.Project, plans []entity.ProjectPlan, images []entity.ProjectImage) error {
	return r.dbCtx(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, txKey{}, tx)
		if err := r.dbCtx(txCtx).Create(p).Error; err != nil {
			return err
		}
		for i := range plans {
			plans[i].ProjectID = p.ID
		}
		if len(plans) > 0 {
			if err := r.dbCtx(txCtx).Create(&plans).Error; err != nil {
				return err
			}
		}
		for i := range images {
			images[i].ProjectID = p.ID
		}
		if len(images) > 0 {
			if err := r.dbCtx(txCtx).Create(&images).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *projectRepo) FindByID(ctx context.Context, id uuid.UUID) (*entity.Project, []entity.ProjectPlan, []entity.ProjectImage, error) {
	var p entity.Project
	err := r.dbCtx(ctx).Where("id = ?", id).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, nil, nil
	}
	if err != nil {
		return nil, nil, nil, err
	}
	var plans []entity.ProjectPlan
	if err := r.dbCtx(ctx).Where("project_id = ?", id).Order("membership_type ASC").Find(&plans).Error; err != nil {
		return nil, nil, nil, err
	}
	var images []entity.ProjectImage
	if err := r.dbCtx(ctx).Where("project_id = ?", id).Order("sort_order ASC").Find(&images).Error; err != nil {
		return nil, nil, nil, err
	}
	return &p, plans, images, nil
}

func (r *projectRepo) ListPublic(ctx context.Context, f ProjectListFilter) ([]entity.Project, int64, error) {
	now := f.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	q := r.dbCtx(ctx).Model(&entity.Project{}).
		Where("status = ?", entity.ProjectStatusActive).
		Where("(duration_type = ? OR (duration_type = ? AND end_date >= ?))",
			entity.DurationUnlimited, entity.DurationLimited, now)
	if s := strings.TrimSpace(f.Search); s != "" {
		like := "%" + s + "%"
		q = q.Where("(title ILIKE ? OR description ILIKE ?)", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	limit, offset := paginate(f.Limit, f.Offset)
	order := publicSort(f.Sort)
	var rows []entity.Project
	err := q.Order(order).Limit(limit).Offset(offset).Find(&rows).Error
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

func (r *projectRepo) ListBySeller(ctx context.Context, sellerID uuid.UUID, limit, offset int) ([]entity.Project, int64, error) {
	q := r.dbCtx(ctx).Model(&entity.Project{}).Where("seller_id = ?", sellerID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	limit, offset = paginate(limit, offset)
	var rows []entity.Project
	err := r.dbCtx(ctx).Where("seller_id = ?", sellerID).
		Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, total, err
}

func (r *projectRepo) ListAll(ctx context.Context, status *entity.ProjectStatus, limit, offset int) ([]entity.Project, int64, error) {
	q := r.dbCtx(ctx).Model(&entity.Project{})
	if status != nil && status.Valid() {
		q = q.Where("status = ?", *status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	limit, offset = paginate(limit, offset)
	var rows []entity.Project
	db := r.dbCtx(ctx).Order("created_at DESC").Limit(limit).Offset(offset)
	if status != nil && status.Valid() {
		db = db.Where("status = ?", *status)
	}
	err := db.Find(&rows).Error
	return rows, total, err
}

func (r *projectRepo) Update(ctx context.Context, p *entity.Project) error {
	return r.dbCtx(ctx).Save(p).Error
}

func (r *projectRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.dbCtx(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, txKey{}, tx)
		if err := r.dbCtx(txCtx).Where("project_id = ?", id).Delete(&entity.ProjectImage{}).Error; err != nil {
			return err
		}
		if err := r.dbCtx(txCtx).Where("project_id = ?", id).Delete(&entity.ProjectPlan{}).Error; err != nil {
			return err
		}
		return r.dbCtx(txCtx).Where("id = ?", id).Delete(&entity.Project{}).Error
	})
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
