package usecase

import (
	"context"
	"strings"
	"time"

	"crowdfund-app/auth"
	"crowdfund-app/entity"
	"crowdfund-app/repository"

	"github.com/google/uuid"
)

func (a *AuthUseCase) RegisterBuyer(ctx context.Context, email, password string, name *string, meta AuthMeta) (*entity.User, error) {
	var (
		normalized  string
		err         error
		exists      bool
		hash        string
		now         time.Time
		trimmedName *string
		n           string
		u           *entity.User
	)

	normalized, err = entity.ValidateBuyerReg(email, password)
	if err != nil {
		switch err {
		case entity.ErrBadEmail:
			return nil, ErrInvalidEmail
		case entity.ErrWeakPassword:
			return nil, ErrWeakPassword
		default:
			return nil, err
		}
	}
	exists, err = a.Users.EmailExists(ctx, normalized)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrEmailInUse
	}
	hash, err = auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	now = time.Now().UTC()
	if name != nil {
		n = strings.TrimSpace(*name)
		if n != "" {
			trimmedName = &n
		}
	}
	u = &entity.User{
		ID:            uuid.New(),
		Email:         normalized,
		Name:          trimmedName,
		PasswordHash:  hash,
		Role:          entity.UserRoleBuyer,
		AccountStatus: entity.AccountStatusPending,
		SellerStatus:  entity.SellerStatusNotApplicable,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err = a.Users.Create(ctx, u); err != nil {
		return nil, err
	}
	a.logActivity(ctx, &u.ID, entity.UserRegister, "user", u.ID.String(), meta, nil)
	return u, nil
}

func (a *AuthUseCase) EmailFree(ctx context.Context, email string) (bool, error) {
	var (
		normalized string
		err        error
		exists     bool
	)

	normalized, err = entity.ValidateEmail(email)
	if err != nil {
		return false, ErrInvalidEmail
	}
	exists, err = a.Users.EmailExists(ctx, normalized)
	if err != nil {
		return false, err
	}
	return !exists, nil
}

func (a *AuthUseCase) Login(ctx context.Context, email, password string, meta AuthMeta) (*LoginResult, error) {
	var (
		normalized string
		u          *entity.User
		err        error
		tokens     *SessionTokens
	)

	normalized = entity.NormalizeEmail(email)
	if normalized == "" || password == "" {
		a.logActivity(ctx, nil, entity.UserLoginFail, "user", normalized, meta, nil)
		return nil, ErrInvalidCredentials
	}
	u, err = a.Users.FindEmail(ctx, normalized)
	if err != nil {
		return nil, err
	}
	if u == nil || auth.ComparePassword(u.PasswordHash, password) != nil {
		a.logActivity(ctx, nil, entity.UserLoginFail, "user", normalized, meta, nil)
		return nil, ErrInvalidCredentials
	}
	if !u.CanLogin() {
		a.logActivity(ctx, &u.ID, entity.UserLoginFail, "user", u.ID.String(), meta, nil)
		return nil, ErrInactiveUser
	}
	if u.NeedsTOTP() && u.TOTPEnabled {
		a.logActivity(ctx, &u.ID, entity.UserLogin, "user", u.ID.String(), meta, entity.StepMeta("2fa_pending"))
		return &LoginResult{User: u, Requires2FA: true}, nil
	}
	tokens, err = a.issueSession(ctx, u, twoFAVerified(u))
	if err != nil {
		return nil, err
	}
	a.logActivity(ctx, &u.ID, entity.UserLogin, "user", u.ID.String(), meta, nil)
	return &LoginResult{
		User:         u,
		AuthToken:    tokens.AuthToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

func (a *AuthUseCase) Refresh(ctx context.Context, refreshTokenPlain string, meta AuthMeta) (string, string, error) {
	var (
		plain  string
		hash   string
		now    time.Time
		row    *entity.RefreshToken
		err    error
		u      *entity.User
		tokens *SessionTokens
	)

	plain = strings.TrimSpace(refreshTokenPlain)
	if plain == "" {
		return "", "", ErrInvalidToken
	}
	// ── TX の外（読み取り・検証）──
	hash = auth.OpaqueHash(plain)
	now = time.Now().UTC()
	row, err = a.RefreshTokens.FindValidHash(ctx, hash, now) // revoked_at IS NULL
	if err != nil || row == nil {
		return "", "", ErrInvalidToken
	}
	u, err = a.Users.FindID(ctx, row.UserID)
	if err != nil || u == nil || !u.CanLogin() {
		return "", "", ErrInvalidToken
	}
	// ── TX の中（原子的にローテーション）──
	err = repository.WithTx(ctx, a.DB, func(txCtx context.Context) error {
		if err = a.RefreshTokens.Revoke(txCtx, row.ID, now); err != nil {
			return err
		} // ① 古い RefreshTokens を失効
		tokens, err = a.issueSession(txCtx, u, twoFAVerified(u)) // ② 新 JWT + 新 RefreshTokens を INSERT
		return err
	})
	if err != nil {
		return "", "", err
	}
	return tokens.AuthToken, tokens.RefreshToken, nil
}

func (a *AuthUseCase) Logout(ctx context.Context, refreshTokenPlain string, meta AuthMeta) error {
	var (
		plain string
		hash  string
		now   time.Time
		row   *entity.RefreshToken
		err   error
	)

	plain = strings.TrimSpace(refreshTokenPlain)
	if plain == "" {
		return ErrInvalidToken
	}
	hash = auth.OpaqueHash(plain)
	now = time.Now().UTC()
	row, err = a.RefreshTokens.FindValidHash(ctx, hash, now)
	if err != nil {
		return err
	}
	if row == nil {
		return ErrInvalidToken
	}
	if err = a.RefreshTokens.Revoke(ctx, row.ID, now); err != nil {
		return err
	}
	a.logActivity(ctx, &row.UserID, entity.UserLogout, "user", row.UserID.String(), meta, nil)
	return nil
}
