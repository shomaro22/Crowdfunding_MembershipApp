package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"crowdfund-app/model"
	"crowdfund-app/repository"
	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type resetRepo struct {
	created *model.ResetToken
	row     model.ResetToken
}

func (r *resetRepo) Create(_ context.Context, row *model.ResetToken) error {
	r.created = row
	return nil
}
func (r *resetRepo) GetValidHash(_ context.Context, hash string, _ time.Time) (model.ResetToken, error) {
	if hash != r.row.TokenHash {
		return model.ResetToken{}, gorm.ErrRecordNotFound
	}
	return r.row, nil
}
func (r *resetRepo) MarkUsed(_ context.Context, _ uuid.UUID, _ time.Time) error { return nil }

type passwordUserRepo struct {
	u model.User
}

func (r *passwordUserRepo) Create(_ context.Context, _ *model.User) error { return nil }
func (r *passwordUserRepo) GetEmail(_ context.Context, email string) (model.User, error) {
	if email != r.u.Email {
		return model.User{}, gorm.ErrRecordNotFound
	}
	return r.u, nil
}
func (r *passwordUserRepo) GetID(_ context.Context, id uuid.UUID) (model.User, error) {
	if id != r.u.ID {
		return model.User{}, gorm.ErrRecordNotFound
	}
	return r.u, nil
}
func (r *passwordUserRepo) Update(_ context.Context, u *model.User) error {
	r.u = *u
	return nil
}
func (r *passwordUserRepo) EmailExists(_ context.Context, _ string) (bool, error) { return false, nil }
func (r *passwordUserRepo) BumpTokenVer(_ context.Context, _ uuid.UUID) (int64, error) {
	return 1, nil
}
func (r *passwordUserRepo) ListRoleStatus(_ context.Context, _ model.UserRole, _ model.AccountStatus, _, _ int) ([]model.User, int64, error) {
	return nil, 0, nil
}
func (r *passwordUserRepo) ListAdmin(_ context.Context, _ repository.AdminUserListFilter) ([]model.User, int64, error) {
	return nil, 0, nil
}

type noopMailer struct{}

func (noopMailer) SendPasswordReset(_ context.Context, _, _ string) error { return nil }

func TestForgotPasswordCreatesResetToken(t *testing.T) {
	var (
		users   passwordUserRepo
		resets  resetRepo
		uid     = uuid.New()
		hash    string
		err     error
		uc      usecase.IAuthUsecase
	)

	hash, err = usecase.HashPassword("Password1!")
	if err != nil {
		t.Fatal(err)
	}
	users.u = model.User{
		ID:            uid,
		Email:         "buyer@example.com",
		PasswordHash:  hash,
		AccountStatus: model.AccountStatusActive,
	}
	uc = usecase.NewAuthUsecase(usecase.AuthConfig{
		Users:     &users,
		ResetToks: &resets,
		Mailer:    noopMailer{},
		JWTSecret: "01234567890123456789012345678901",
	})

	err = uc.ForgotPassword(context.Background(), users.u.Email)
	if err != nil {
		t.Fatalf("forgot: %v", err)
	}
	if resets.created == nil || resets.created.UserID != uid {
		t.Fatal("expected reset token created")
	}
}

func TestChangePasswordRejectsWrongCurrent(t *testing.T) {
	var (
		users  passwordUserRepo
		uid    = uuid.New()
		hash   string
		err    error
		uc     usecase.IAuthUsecase
	)

	hash, err = usecase.HashPassword("Password1!")
	if err != nil {
		t.Fatal(err)
	}
	users.u = model.User{
		ID:            uid,
		Email:         "buyer@example.com",
		PasswordHash:  hash,
		Role:          model.UserRoleBuyer,
		AccountStatus: model.AccountStatusActive,
	}
	uc = usecase.NewAuthUsecase(usecase.AuthConfig{
		Users:     &users,
		JWTSecret: "01234567890123456789012345678901",
	})

	err = uc.ChangePassword(context.Background(), uid, "WrongPass1!", "NewPassword1!", "")
	if !errors.Is(err, usecase.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestGetUserReturnsView(t *testing.T) {
	var (
		users passwordUserRepo
		uid   = uuid.New()
		name  = "Buyer"
		uc    = usecase.NewAuthUsecase(usecase.AuthConfig{
			Users:     &users,
			JWTSecret: "01234567890123456789012345678901",
		})
		view usecase.UserView
		err  error
	)

	users.u = model.User{
		ID:            uid,
		Email:         "buyer@example.com",
		Name:          &name,
		Role:          model.UserRoleBuyer,
		AccountStatus: model.AccountStatusActive,
	}
	view, err = uc.GetUser(context.Background(), uid)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if view.Email != users.u.Email {
		t.Fatalf("email=%q", view.Email)
	}
}

func TestOAuthDisabledByDefault(t *testing.T) {
	var uc = usecase.NewAuthUsecase(usecase.AuthConfig{
		JWTSecret: "01234567890123456789012345678901",
	})
	if uc.OAuthEnabled() {
		t.Fatal("oauth should be disabled without env")
	}
	_, err := uc.GoogleAuthURL("/", usecase.RequestMeta{})
	if !errors.Is(err, usecase.ErrOAuthUnavailable) {
		t.Fatalf("expected ErrOAuthUnavailable, got %v", err)
	}
}
