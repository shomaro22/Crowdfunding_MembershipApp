package repository

import (
	"context"
	"strings"
	"time"

	"crowdfund-app/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IProjectRepo interface {
	Create(ctx context.Context, p *model.Project, plans []model.ProjectPlan, images []model.ProjectImage) error
	GetID(ctx context.Context, id uuid.UUID) (model.Project, []model.ProjectPlan, []model.ProjectImage, error)
	ListPublic(ctx context.Context, f ProjectListFilter) ([]model.Project, int64, error)
	ListFeatured(ctx context.Context, limit int, asOf time.Time) ([]model.Project, error)
	ListSeller(ctx context.Context, sellerID uuid.UUID, limit, offset int) ([]model.Project, int64, error)
	ListAll(ctx context.Context, status *model.ProjectStatus, limit, offset int) ([]model.Project, int64, error)
	Update(ctx context.Context, p *model.Project) error
	ReplaceImages(ctx context.Context, projectID uuid.UUID, images []model.ProjectImage) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetPlan(ctx context.Context, projectID, planID uuid.UUID) (model.ProjectPlan, error)
	IncrementSupportCounts(ctx context.Context, projectID, planID uuid.UUID) error
	TransferPlanSupportCount(ctx context.Context, projectID, fromPlanID, toPlanID uuid.UUID) error
	AddCurrentAmount(ctx context.Context, projectID uuid.UUID, amount int64) error
	ListPlansByProjectIDs(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID][]model.ProjectPlan, error)
	ListImagesByProjectIDs(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID][]model.ProjectImage, error)
}

type projectRepo struct {
	db *gorm.DB
}

func NewProjectRepo(db *gorm.DB) IProjectRepo {
	return &projectRepo{db: db}
}

func (r *projectRepo) dbCtx(ctx context.Context) *gorm.DB {
	return dbFromCtx(ctx, r.db)
}

func (r *projectRepo) Create(ctx context.Context, p *model.Project, plans []model.ProjectPlan, images []model.ProjectImage) error {
	return r.dbCtx(ctx).Transaction(func(tx *gorm.DB) error {
		var (
			txCtx context.Context
			err   error
			i     int
		)
		txCtx = context.WithValue(ctx, txKey{}, tx)
		err = r.dbCtx(txCtx).Create(p).Error
		if err != nil {
			return err
		}
		for i = range plans {
			plans[i].ProjectID = p.ID
		}
		if len(plans) > 0 {
			err = r.dbCtx(txCtx).Create(&plans).Error
			if err != nil {
				return err
			}
		}
		for i = range images {
			images[i].ProjectID = p.ID
		}
		if len(images) > 0 {
			err = r.dbCtx(txCtx).Create(&images).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *projectRepo) Update(ctx context.Context, p *model.Project) error {
	return r.dbCtx(ctx).Save(p).Error
}

func (r *projectRepo) ReplaceImages(ctx context.Context, projectID uuid.UUID, images []model.ProjectImage) error {
	return r.dbCtx(ctx).Transaction(func(tx *gorm.DB) error {
		var (
			txCtx context.Context
			err   error
			i     int
		)
		txCtx = context.WithValue(ctx, txKey{}, tx)
		err = r.dbCtx(txCtx).Where("project_id = ?", projectID).Delete(&model.ProjectImage{}).Error
		if err != nil {
			return err
		}
		for i = range images {
			images[i].ProjectID = projectID
		}
		if len(images) > 0 {
			return r.dbCtx(txCtx).Create(&images).Error
		}
		return nil
	})
}

func (r *projectRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.dbCtx(ctx).Transaction(func(tx *gorm.DB) error {
		var (
			txCtx context.Context
			err   error
		)
		txCtx = context.WithValue(ctx, txKey{}, tx)
		err = r.dbCtx(txCtx).Where("project_id = ?", id).Delete(&model.ProjectImage{}).Error
		if err != nil {
			return err
		}
		err = r.dbCtx(txCtx).Where("project_id = ?", id).Delete(&model.ProjectPlan{}).Error
		if err != nil {
			return err
		}
		return r.dbCtx(txCtx).Where("id = ?", id).Delete(&model.Project{}).Error
	})
}

func (r *projectRepo) IncrementSupportCounts(ctx context.Context, projectID, planID uuid.UUID) error {
	var err error
	err = r.dbCtx(ctx).Model(&model.Project{}).Where("id = ?", projectID).
		UpdateColumn("supporter_count", gorm.Expr("supporter_count + 1")).Error
	if err != nil {
		return err
	}
	return r.dbCtx(ctx).Model(&model.ProjectPlan{}).Where("id = ? AND project_id = ?", planID, projectID).
		UpdateColumn("supporter_count", gorm.Expr("supporter_count + 1")).Error
}

func (r *projectRepo) TransferPlanSupportCount(ctx context.Context, projectID, fromPlanID, toPlanID uuid.UUID) error {
	var err error
	if fromPlanID == toPlanID {
		return nil
	}
	err = r.dbCtx(ctx).Model(&model.ProjectPlan{}).
		Where("id = ? AND project_id = ? AND supporter_count > 0", fromPlanID, projectID).
		UpdateColumn("supporter_count", gorm.Expr("supporter_count - 1")).Error
	if err != nil {
		return err
	}
	return r.dbCtx(ctx).Model(&model.ProjectPlan{}).Where("id = ? AND project_id = ?", toPlanID, projectID).
		UpdateColumn("supporter_count", gorm.Expr("supporter_count + 1")).Error
}

func (r *projectRepo) AddCurrentAmount(ctx context.Context, projectID uuid.UUID, amount int64) error {
	if amount <= 0 {
		return nil
	}
	return r.dbCtx(ctx).Model(&model.Project{}).Where("id = ?", projectID).
		UpdateColumn("current_amount", gorm.Expr("current_amount + ?", amount)).Error
}

type ProjectListFilter struct {
	Search string
	Sort   string
	Limit  int
	Offset int
	AsOf   time.Time
}

func (r *projectRepo) GetID(ctx context.Context, id uuid.UUID) (model.Project, []model.ProjectPlan, []model.ProjectImage, error) {
	var (
		p      model.Project
		plans  []model.ProjectPlan
		images []model.ProjectImage
		err    error
	)

	err = firstOne(r.dbCtx(ctx).Where("id = ?", id), &p)
	if err != nil {
		return model.Project{}, nil, nil, err
	}
	if err = r.dbCtx(ctx).Where("project_id = ?", id).Order("membership_type ASC").Find(&plans).Error; err != nil {
		return model.Project{}, nil, nil, err
	}
	if err = r.dbCtx(ctx).Where("project_id = ?", id).Order("sort_order ASC").Find(&images).Error; err != nil {
		return model.Project{}, nil, nil, err
	}
	return p, plans, images, nil
}

func (r *projectRepo) ListPublic(ctx context.Context, f ProjectListFilter) ([]model.Project, int64, error) {
	var (
		asOf   time.Time
		q      *gorm.DB
		total  int64
		limit  int
		offset int
		order  string
		rows   []model.Project
		s      string
		like   string
		err    error
	)

	asOf = f.AsOf
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	q = r.dbCtx(ctx).Model(&model.Project{}).
		Where("status = ?", model.ProjectStatusActive).
		Where("(duration_type = ? OR (duration_type = ? AND end_date >= ?))",
			model.DurationUnlimited, model.DurationLimited, asOf)
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

func (r *projectRepo) ListFeatured(ctx context.Context, limit int, asOf time.Time) ([]model.Project, error) {
	var (
		rows []model.Project
		err  error
	)
	if limit <= 0 {
		limit = 5
	}
	if limit > 20 {
		limit = 20
	}
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	err = r.dbCtx(ctx).Model(&model.Project{}).
		Where("status = ?", model.ProjectStatusActive).
		Where("(duration_type = ? OR (duration_type = ? AND end_date >= ?))",
			model.DurationUnlimited, model.DurationLimited, asOf).
		Order("current_amount DESC, supporter_count DESC, created_at DESC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
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

func (r *projectRepo) ListSeller(ctx context.Context, sellerID uuid.UUID, limit, offset int) ([]model.Project, int64, error) {
	var (
		q     *gorm.DB
		total int64
		rows  []model.Project
		err   error
	)
	q = r.dbCtx(ctx).Model(&model.Project{}).Where("seller_id = ?", sellerID)
	err = q.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	limit, offset = paginate(limit, offset)
	err = r.dbCtx(ctx).Where("seller_id = ?", sellerID).
		Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, total, err
}

func (r *projectRepo) ListAll(ctx context.Context, status *model.ProjectStatus, limit, offset int) ([]model.Project, int64, error) {
	var (
		q     *gorm.DB
		total int64
		rows  []model.Project
		db    *gorm.DB
		err   error
	)
	q = r.dbCtx(ctx).Model(&model.Project{})
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

func (r *projectRepo) GetPlan(ctx context.Context, projectID, planID uuid.UUID) (model.ProjectPlan, error) {
	return getOne[model.ProjectPlan](
		r.dbCtx(ctx).Where("id = ? AND project_id = ?", planID, projectID),
	)
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

func (r *projectRepo) ListPlansByProjectIDs(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID][]model.ProjectPlan, error) {
	var (
		rows []model.ProjectPlan
		out  map[uuid.UUID][]model.ProjectPlan
		i    int
		err  error
	)
	out = map[uuid.UUID][]model.ProjectPlan{}
	if len(projectIDs) == 0 {
		return out, nil
	}
	err = r.dbCtx(ctx).Where("project_id IN ?", projectIDs).Order("membership_type ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for i = range rows {
		pid := rows[i].ProjectID
		out[pid] = append(out[pid], rows[i])
	}
	return out, nil
}

func (r *projectRepo) ListImagesByProjectIDs(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID][]model.ProjectImage, error) {
	var (
		rows []model.ProjectImage
		out  map[uuid.UUID][]model.ProjectImage
		i    int
		err  error
	)
	out = map[uuid.UUID][]model.ProjectImage{}
	if len(projectIDs) == 0 {
		return out, nil
	}
	err = r.dbCtx(ctx).Where("project_id IN ?", projectIDs).Order("sort_order ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for i = range rows {
		pid := rows[i].ProjectID
		out[pid] = append(out[pid], rows[i])
	}
	return out, nil
}
