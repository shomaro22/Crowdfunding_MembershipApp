package repository

import (
	"context"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepo interface {
	Create(ctx context.Context, u *entity.User) error
	Update(ctx context.Context, u *entity.User) error
	FindID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	FindEmail(ctx context.Context, email string) (*entity.User, error)
	EmailExists(ctx context.Context, email string) (bool, error)
	ListRoleStatus(ctx context.Context, role entity.UserRole, status entity.AccountStatus, limit, offset int) ([]entity.User, int64, error)
	ListAdmin(ctx context.Context, f AdminUserListFilter) ([]entity.User, int64, error)
	BumpTokenVer(ctx context.Context, userID uuid.UUID) (int64, error)
}

type userRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) UserRepo {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, u *entity.User) error {
	return dbFromCtx(ctx, r.db).Create(u).Error
}

func (r *userRepo) Update(ctx context.Context, u *entity.User) error {
	return dbFromCtx(ctx, r.db).Save(u).Error
}

func (r *userRepo) FindID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	return firstOnePtr[entity.User](dbFromCtx(ctx, r.db).Where("id = ?", id))
}

func (r *userRepo) FindEmail(ctx context.Context, email string) (*entity.User, error) {
	return firstOnePtr[entity.User](dbFromCtx(ctx, r.db).Where("email = ?", email))
}

func (r *userRepo) EmailExists(ctx context.Context, email string) (bool, error) {
	var (
		count int64
		err   error
	)
	err = dbFromCtx(ctx, r.db).Model(&entity.User{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

func (r *userRepo) ListRoleStatus(ctx context.Context, role entity.UserRole, status entity.AccountStatus, limit, offset int) ([]entity.User, int64, error) {
	var (
		q     *gorm.DB
		total int64
		rows  []entity.User
		err   error
	)
	q = dbFromCtx(ctx, r.db).Model(&entity.User{}).Where("role = ? AND account_status = ?", role, status)
	err = q.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	err = dbFromCtx(ctx, r.db).Where("role = ? AND account_status = ?", role, status).
		Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, total, err
}

func (r *userRepo) ListAdmin(ctx context.Context, f AdminUserListFilter) ([]entity.User, int64, error) {
	var (
		q     *gorm.DB
		total int64
		rows  []entity.User
		err   error
	)
	q = dbFromCtx(ctx, r.db).Model(&entity.User{})
	if f.Role != nil {
		q = q.Where("role = ?", *f.Role)
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("email ILIKE ? OR name ILIKE ?", like, like)
	}
	err = q.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	limit, offset := paginate(f.Limit, f.Offset)
	err = q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, total, err
}

func (r *userRepo) BumpTokenVer(ctx context.Context, userID uuid.UUID) (int64, error) {
	var (
		res *gorm.DB
		u   entity.User
		err error
	)
	res = dbFromCtx(ctx, r.db).Model(&entity.User{}).
		Where("id = ?", userID).
		UpdateColumn("token_version", gorm.Expr("token_version + 1"))
	if res.Error != nil {
		return 0, res.Error
	}
	err = dbFromCtx(ctx, r.db).Select("token_version").Where("id = ?", userID).First(&u).Error
	if err != nil {
		return 0, err
	}
	return u.TokenVersion, nil
}