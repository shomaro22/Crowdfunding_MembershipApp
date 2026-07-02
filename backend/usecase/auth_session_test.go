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

type recordingUserRepo struct {
	getEmailCalls int
	getIDCalls    int
	emailUser     model.User
	idUser        model.User
}

func (r *recordingUserRepo) Create(_ context.Context, _ *model.User) error { return nil }
func (r *recordingUserRepo) GetEmail(_ context.Context, email string) (model.User, error) {
	r.getEmailCalls++
	if email != r.emailUser.Email {
		return model.User{}, gorm.ErrRecordNotFound
	}
	return r.emailUser, nil
}
func (r *recordingUserRepo) GetID(_ context.Context, id uuid.UUID) (model.User, error) {
	r.getIDCalls++
	if id != r.idUser.ID {
		return model.User{}, gorm.ErrRecordNotFound
	}
	return r.idUser, nil
}
func (r *recordingUserRepo) Update(_ context.Context, _ *model.User) error { return nil }
func (r *recordingUserRepo) EmailExists(_ context.Context, _ string) (bool, error) {
	return false, nil
}
func (r *recordingUserRepo) BumpTokenVer(_ context.Context, _ uuid.UUID) (int64, error) { return 0, nil }
func (r *recordingUserRepo) ListRoleStatus(_ context.Context, _ model.UserRole, _ model.AccountStatus, _, _ int) ([]model.User, int64, error) {
	return nil, 0, nil
}
func (r *recordingUserRepo) ListAdmin(_ context.Context, _ repository.AdminUserListFilter) ([]model.User, int64, error) {
	return nil, 0, nil
}

type recordingRefreshRepo struct {
	createCalls int
	revokeCalls int
	getCalls    int
	lastHash    string
	validRow    model.RefreshToken
}

func (r *recordingRefreshRepo) Create(_ context.Context, _ *model.RefreshToken) error {
	r.createCalls++
	return nil
}
func (r *recordingRefreshRepo) GetValidHash(_ context.Context, hash string, _ time.Time) (model.RefreshToken, error) {
	r.getCalls++
	r.lastHash = hash
	if hash != r.validRow.TokenHash {
		return model.RefreshToken{}, gorm.ErrRecordNotFound
	}
	return r.validRow, nil
}
func (r *recordingRefreshRepo) GetByHash(_ context.Context, hash string) (model.RefreshToken, error) {
	if hash != r.validRow.TokenHash {
		return model.RefreshToken{}, gorm.ErrRecordNotFound
	}
	return r.validRow, nil
}

func (r *recordingRefreshRepo) Revoke(_ context.Context, id uuid.UUID, _ time.Time) error {
	r.revokeCalls++
	if id != r.validRow.ID {
		return errors.New("unexpected revoke id")
	}
	return nil
}
func (r *recordingRefreshRepo) RevokeAllByUser(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}

type recordingActivityRepo struct {
	createCalls int
	lastAction  model.ActivityAction
}

func (r *recordingActivityRepo) Create(_ context.Context, row *model.ActivityLog) error {
	r.createCalls++
	r.lastAction = row.Action
	return nil
}
func (r *recordingActivityRepo) List(_ context.Context, _ repository.ActivityListFilter) ([]model.ActivityLog, int64, error) {
	return nil, 0, nil
}
func (r *recordingActivityRepo) CountByActionBetween(_ context.Context, _ model.ActivityAction, _, _ time.Time) (int64, error) {
	return 0, nil
}
func (r *recordingActivityRepo) TopIPsBetween(_ context.Context, _, _ time.Time, _ int) ([]repository.ActivityIPCount, error) {
	return nil, nil
}

func TestAuthLoginCallsUserRepoAndActivity(t *testing.T) {
	var (
		users    recordingUserRepo
		refresh  recordingRefreshRepo
		activity recordingActivityRepo
		uc      usecase.AuthUseCase
		hash     string
		res      *usecase.LoginResult
		err      error
		uid      uuid.UUID
		meta    usecase.RequestMeta
	)
	uid = uuid.New()
	hash, err =usecase.HashPassword("Password1!")
	if err != nil {
		t.Fatal(err)
	}
	users.emailUser = model.User{
		ID:           uid,
		Email:        "buyer@example.com",
		PasswordHash: hash,
		Role:         model.UserRoleBuyer,
		AccountStatus: model.AccountStatusActive,
		TokenVersion: 1,
	}
	users.idUser = users.emailUser
	uc =usecase.AuthUseCase{
		Users:         &users,
		RefreshTokens: &refresh,
		Activity:      &activity,
		Tokens:       usecase.NewTokenService([]byte("01234567890123456789012345678901"),usecase.DefaultAuthTTL),
	}
	meta =usecase.RequestMeta{IPAddress: "127.0.0.1", UserAgent: "test", RequestID: "req-1"}

	res, err = uc.Login(context.Background(), users.emailUser.Email, "Password1!", meta)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if res.AuthToken == "" || res.RefreshToken == "" {
		t.Fatal("expected tokens")
	}
	if users.getEmailCalls != 1 {
		t.Fatalf("GetEmail calls=%d want 1", users.getEmailCalls)
	}
	if refresh.createCalls != 1 {
		t.Fatalf("Refresh Create calls=%d want 1", refresh.createCalls)
	}
	if activity.createCalls != 1 {
		t.Fatalf("Activity Create calls=%d want 1", activity.createCalls)
	}
	if activity.lastAction != model.UserLogin {
		t.Fatalf("activity action=%q want %q", activity.lastAction, model.UserLogin)
	}
}

func TestAuthLoginInvalidPasswordDoesNotIssueSession(t *testing.T) {
	var (
		users    recordingUserRepo
		refresh  recordingRefreshRepo
		activity recordingActivityRepo
		uc      usecase.AuthUseCase
		hash     string
		err      error
		uid      uuid.UUID
	)
	uid = uuid.New()
	hash, err =usecase.HashPassword("Password1!")
	if err != nil {
		t.Fatal(err)
	}
	users.emailUser = model.User{
		ID:           uid,
		Email:        "buyer@example.com",
		PasswordHash: hash,
		Role:         model.UserRoleBuyer,
		AccountStatus: model.AccountStatusActive,
	}
	uc =usecase.AuthUseCase{
		Users:         &users,
		RefreshTokens: &refresh,
		Activity:      &activity,
		Tokens:       usecase.NewTokenService([]byte("01234567890123456789012345678901"),usecase.DefaultAuthTTL),
	}

	_, err = uc.Login(context.Background(), users.emailUser.Email, "WrongPass1!",usecase.RequestMeta{})
	if !errors.Is(err, usecase.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
	if refresh.createCalls != 0 {
		t.Fatalf("Refresh Create calls=%d want 0", refresh.createCalls)
	}
	if activity.createCalls != 1 {
		t.Fatalf("login fail activity calls=%d want 1", activity.createCalls)
	}
	if activity.lastAction != model.UserLoginFail {
		t.Fatalf("activity action=%q want login_fail", activity.lastAction)
	}
}

func TestAuthLogoutRevokesRefreshAndLogsActivity(t *testing.T) {
	var (
		refresh  recordingRefreshRepo
		activity recordingActivityRepo
		uc      usecase.AuthUseCase
		plain    string
		uid      uuid.UUID
		meta    usecase.RequestMeta
		err      error
	)
	uid = uuid.New()
	plain = "logout-plain-token"
	refresh.validRow = model.RefreshToken{
		ID:        uuid.New(),
		UserID:    uid,
		TokenHash:usecase.OpaqueHash(plain),
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}
	uc =usecase.AuthUseCase{
		RefreshTokens: &refresh,
		Activity:      &activity,
	}
	meta =usecase.RequestMeta{IPAddress: "127.0.0.1", UserAgent: "test", RequestID: "req-2"}

	err = uc.Logout(context.Background(), plain, meta)
	if err != nil {
		t.Fatalf("logout: %v", err)
	}
	if refresh.getCalls != 1 {
		t.Fatalf("GetValidHash calls=%d want 1", refresh.getCalls)
	}
	if refresh.revokeCalls != 1 {
		t.Fatalf("Revoke calls=%d want 1", refresh.revokeCalls)
	}
	if activity.createCalls != 1 {
		t.Fatalf("activity calls=%d want 1", activity.createCalls)
	}
	if activity.lastAction != model.UserLogout {
		t.Fatalf("activity action=%q want logout", activity.lastAction)
	}
}

func TestAuthLogoutEmptyTokenDoesNotTouchRepo(t *testing.T) {
	var (
		refresh  recordingRefreshRepo
		activity recordingActivityRepo
		uc      usecase.AuthUseCase
	)
	uc =usecase.AuthUseCase{
		RefreshTokens: &refresh,
		Activity:      &activity,
	}
	if err := uc.Logout(context.Background(), "  ", usecase.RequestMeta{}); !errors.Is(err, usecase.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
	if refresh.getCalls != 0 || refresh.revokeCalls != 0 {
		t.Fatal("refresh repo should not be called for empty token")
	}
	if activity.createCalls != 0 {
		t.Fatal("activity should not be logged for invalid logout")
	}
}
