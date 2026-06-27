package repository

import (
	"context"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProjectRepo interface {
	Create(ctx context.Context, p *entity.Project, plans []entity.ProjectPlan, images []entity.ProjectImage) error
	FindID(ctx context.Context, id uuid.UUID) (*entity.Project, []entity.ProjectPlan, []entity.ProjectImage, error)
	ListPublic(ctx context.Context, f ProjectListFilter) ([]entity.Project, int64, error)
	ListSeller(ctx context.Context, sellerID uuid.UUID, limit, offset int) ([]entity.Project, int64, error)
	ListAll(ctx context.Context, status *entity.ProjectStatus, limit, offset int) ([]entity.Project, int64, error)
	Update(ctx context.Context, p *entity.Project) error
	ReplaceImages(ctx context.Context, projectID uuid.UUID, images []entity.ProjectImage) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindPlan(ctx context.Context, projectID, planID uuid.UUID) (*entity.ProjectPlan, error)
	IncrementSupportCounts(ctx context.Context, projectID, planID uuid.UUID) error
	TransferPlanSupportCount(ctx context.Context, projectID, fromPlanID, toPlanID uuid.UUID) error
	AddCurrentAmount(ctx context.Context, projectID uuid.UUID, amount int64) error
}

type projectRepo struct {
	db *gorm.DB
}

func NewProjectRepo(db *gorm.DB) ProjectRepo {
	return &projectRepo{db: db}
}

func (r *projectRepo) dbCtx(ctx context.Context) *gorm.DB {
	return dbFromCtx(ctx, r.db)
}

func (r *projectRepo) Create(ctx context.Context, p *entity.Project, plans []entity.ProjectPlan, images []entity.ProjectImage) error {
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

func (r *projectRepo) Update(ctx context.Context, p *entity.Project) error {
	return r.dbCtx(ctx).Save(p).Error
}

func (r *projectRepo) ReplaceImages(ctx context.Context, projectID uuid.UUID, images []entity.ProjectImage) error {
	return r.dbCtx(ctx).Transaction(func(tx *gorm.DB) error {
		var (
			txCtx context.Context
			err   error
			i     int
		)
		txCtx = context.WithValue(ctx, txKey{}, tx)
		err = r.dbCtx(txCtx).Where("project_id = ?", projectID).Delete(&entity.ProjectImage{}).Error
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
		err = r.dbCtx(txCtx).Where("project_id = ?", id).Delete(&entity.ProjectImage{}).Error
		if err != nil {
			return err
		}
		err = r.dbCtx(txCtx).Where("project_id = ?", id).Delete(&entity.ProjectPlan{}).Error
		if err != nil {
			return err
		}
		return r.dbCtx(txCtx).Where("id = ?", id).Delete(&entity.Project{}).Error
	})
}

func (r *projectRepo) IncrementSupportCounts(ctx context.Context, projectID, planID uuid.UUID) error {
	var err error
	err = r.dbCtx(ctx).Model(&entity.Project{}).Where("id = ?", projectID).
		UpdateColumn("supporter_count", gorm.Expr("supporter_count + 1")).Error
	if err != nil {
		return err
	}
	return r.dbCtx(ctx).Model(&entity.ProjectPlan{}).Where("id = ? AND project_id = ?", planID, projectID).
		UpdateColumn("supporter_count", gorm.Expr("supporter_count + 1")).Error
}

func (r *projectRepo) TransferPlanSupportCount(ctx context.Context, projectID, fromPlanID, toPlanID uuid.UUID) error {
	var err error
	if fromPlanID == toPlanID {
		return nil
	}
	err = r.dbCtx(ctx).Model(&entity.ProjectPlan{}).
		Where("id = ? AND project_id = ? AND supporter_count > 0", fromPlanID, projectID).
		UpdateColumn("supporter_count", gorm.Expr("supporter_count - 1")).Error
	if err != nil {
		return err
	}
	return r.dbCtx(ctx).Model(&entity.ProjectPlan{}).Where("id = ? AND project_id = ?", toPlanID, projectID).
		UpdateColumn("supporter_count", gorm.Expr("supporter_count + 1")).Error
}

func (r *projectRepo) AddCurrentAmount(ctx context.Context, projectID uuid.UUID, amount int64) error {
	if amount <= 0 {
		return nil
	}
	return r.dbCtx(ctx).Model(&entity.Project{}).Where("id = ?", projectID).
		UpdateColumn("current_amount", gorm.Expr("current_amount + ?", amount)).Error
}
