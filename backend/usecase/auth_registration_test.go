package usecase_test

import (
	"context"
	"errors"
	"testing"

	"crowdfund-app/model"
	"crowdfund-app/repository"
	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type regUserRepo struct {
	exists  bool
	created *model.User
}

func (r *regUserRepo) Create(_ context.Context, u *model.User) error {
	r.created = u
	return nil
}
func (r *regUserRepo) GetEmail(_ context.Context, _ string) (model.User, error) {
	return model.User{}, gorm.ErrRecordNotFound
}
func (r *regUserRepo) GetID(_ context.Context, _ uuid.UUID) (model.User, error) {
	return model.User{}, gorm.ErrRecordNotFound
}
func (r *regUserRepo) Update(_ context.Context, _ *model.User) error { return nil }
func (r *regUserRepo) EmailExists(_ context.Context, _ string) (bool, error) {
	return r.exists, nil
}
func (r *regUserRepo) BumpTokenVer(_ context.Context, _ uuid.UUID) (int64, error) { return 0, nil }
func (r *regUserRepo) ListRoleStatus(_ context.Context, _ model.UserRole, _ model.AccountStatus, _, _ int) ([]model.User, int64, error) {
	return nil, 0, nil
}
func (r *regUserRepo) ListAdmin(_ context.Context, _ repository.AdminUserListFilter) ([]model.User, int64, error) {
	return nil, 0, nil
}
func (r *regUserRepo) ListByIDs(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]model.User, error) {
	return map[uuid.UUID]model.User{}, nil
}

func TestRegisterBuyerCreatesActiveBuyer(t *testing.T) {
	var (
		users regUserRepo
		uc    = usecase.NewAuthUsecase(usecase.AuthConfig{
			Users:     &users,
			JWTSecret: "01234567890123456789012345678901",
		})
		view usecase.UserView
		err  error
	)

	view, err = uc.RegisterBuyer(context.Background(), "buyer@example.com", "Password1!", nil, usecase.RequestMeta{})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if view.Email != "buyer@example.com" {
		t.Fatalf("email=%q", view.Email)
	}
	if users.created == nil {
		t.Fatal("expected user create")
	}
	if users.created.AccountStatus != model.AccountStatusActive {
		t.Fatalf("status=%q want active", users.created.AccountStatus)
	}
	if users.created.Role != model.UserRoleBuyer {
		t.Fatalf("role=%q want buyer", users.created.Role)
	}
}

func TestRegisterBuyerRejectsDuplicateEmail(t *testing.T) {
	var (
		users = regUserRepo{exists: true}
		uc    = usecase.NewAuthUsecase(usecase.AuthConfig{
			Users:     &users,
			JWTSecret: "01234567890123456789012345678901",
		})
		_, err = uc.RegisterBuyer(context.Background(), "buyer@example.com", "Password1!", nil, usecase.RequestMeta{})
	)

	if !errors.Is(err, usecase.ErrEmailInUse) {
		t.Fatalf("expected ErrEmailInUse, got %v", err)
	}
}

func TestEmailFreeReturnsTrueForUnused(t *testing.T) {
	var (
		users regUserRepo
		uc    = usecase.NewAuthUsecase(usecase.AuthConfig{
			Users:     &users,
			JWTSecret: "01234567890123456789012345678901",
		})
		free bool
		err  error
	)

	free, err = uc.EmailFree(context.Background(), "new@example.com")
	if err != nil {
		t.Fatalf("email free: %v", err)
	}
	if !free {
		t.Fatal("expected free email")
	}
}

func TestEmailFreeReturnsFalseForUsed(t *testing.T) {
	var (
		users = regUserRepo{exists: true}
		uc    = usecase.NewAuthUsecase(usecase.AuthConfig{
			Users:     &users,
			JWTSecret: "01234567890123456789012345678901",
		})
		free bool
		err  error
	)

	free, err = uc.EmailFree(context.Background(), "used@example.com")
	if err != nil {
		t.Fatalf("email free: %v", err)
	}
	if free {
		t.Fatal("expected used email")
	}
}

func TestValidateSessionRejectsRevokedToken(t *testing.T) {
	var (
		users recordingUserRepo
		uc    = newTestAuth(&users, &recordingRefreshRepo{}, &recordingActivityRepo{})
		uid   = uuid.New()
		claims usecase.AuthClaims
		err    error
	)

	users.idUser = model.User{
		ID:            uid,
		Role:          model.UserRoleBuyer,
		AccountStatus: model.AccountStatusActive,
		TokenVersion:  2,
	}
	claims = usecase.AuthClaims{UserID: uid, TokenVersion: 1}

	_, err = uc.ValidateSession(context.Background(), claims)
	if !errors.Is(err, usecase.ErrTokenRevoked) {
		t.Fatalf("expected ErrTokenRevoked, got %v", err)
	}
}
