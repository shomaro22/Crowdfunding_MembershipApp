package usecase

import (
	"context"
	"time"

	"crowdfund-app/model"

	"github.com/google/uuid"
)

type TOTPSetupResult struct {
	Secret     string
	OTPAuthURL string
}

type TOTPVerifyResult struct {
	BackupCodes []string
}

func (t *TwoFactorUseCase) Setup2FA(ctx context.Context, userID uuid.UUID) (*TOTPSetupResult, error) {
	var (
		u      model.User
		err    error
		issuer string
		setup  *TOTPSetup
		enc    string
	)

	u, err = t.deps.Users.GetID(ctx, userID)
	if err != nil {
		return nil, mapRepoNotFound(err)
	}
	if u.TOTPEnabled {
		return nil, Err2FAAlreadyEnabled
	}
	issuer = t.deps.TOTPIssuer
	if issuer == "" {
		issuer = "Crowdfund"
	}
	setup, err = NewTOTPSetup(issuer, u.Email)
	if err != nil {
		return nil, err
	}
	enc, err = EncryptSecret(t.deps.TOTPKey, setup.Secret)
	if err != nil {
		return nil, err
	}
	u.TOTPSecret = &enc
	u.UpdatedAt = time.Now().UTC()
	if err = t.deps.Users.Update(ctx, &u); err != nil {
		return nil, err
	}
	return &TOTPSetupResult{Secret: setup.Secret, OTPAuthURL: setup.OTPAuthURL}, nil
}

func (t *TwoFactorUseCase) Verify2FASetup(ctx context.Context, userID uuid.UUID, code string) (*TOTPVerifyResult, error) {
	var (
		u      model.User
		err    error
		secret string
		now    time.Time
		codes  []string
	)

	u, err = t.deps.Users.GetID(ctx, userID)
	if err != nil {
		return nil, mapRepoNotFound(err)
	}
	if u.TOTPEnabled {
		return nil, Err2FAAlreadyEnabled
	}
	secret, err = t.deps.decryptTOTP(u)
	if err != nil {
		return nil, Err2FANotConfigured
	}
	if !VerifyTOTP(secret, code) {
		return nil, ErrInvalid2FACode
	}
	now = time.Now().UTC()
	u.TOTPEnabled = true
	u.UpdatedAt = now
	if err = t.deps.Users.Update(ctx, &u); err != nil {
		return nil, err
	}
	codes, err = t.deps.genBackupCodes(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	if err = InvalidateUserSessions(ctx, t.deps.Tx, t.deps.Users, t.deps.RefreshTokens, u.ID); err != nil {
		return nil, err
	}
	return &TOTPVerifyResult{BackupCodes: codes}, nil
}

func (t *TwoFactorUseCase) CreatePending2FAToken(_ context.Context, userID uuid.UUID) (string, error) {
	return SignPending2FA(t.deps.pendingSecret, userID, time.Now().UTC()), nil
}

func (t *TwoFactorUseCase) Challenge2FA(ctx context.Context, pendingToken, code string, meta RequestMeta) (*LoginResult, error) {
	var (
		uid    uuid.UUID
		err    error
		u      model.User
		tokens *SessionTokens
	)

	uid, err = VerifyPending2FA(t.deps.pendingSecret, pendingToken, time.Now().UTC())
	if err != nil {
		return nil, ErrInvalidToken
	}
	u, err = t.deps.Users.GetID(ctx, uid)
	if err != nil {
		return nil, mapRepoNotFoundAs(err, ErrInvalidToken)
	}
	if !u.CanLogin() {
		return nil, ErrInvalidToken
	}
	if !u.TOTPEnabled {
		return nil, Err2FANotConfigured
	}
	_, err = t.deps.decryptTOTP(u)
	if err != nil {
		return nil, Err2FANotConfigured
	}
	if !t.deps.verifyTOTPOrBackup(ctx, u, code) {
		t.deps.logActivity(ctx, &u.ID, model.UserLoginFail, "user", u.ID.String(), meta, model.StepMeta("2fa_fail"))
		return nil, ErrInvalid2FACode
	}
	tokens, err = t.deps.issueSession(ctx, u, true)
	if err != nil {
		return nil, err
	}
	t.deps.logActivity(ctx, &u.ID, model.UserLogin, "user", u.ID.String(), meta, model.StepMeta("2fa_ok"))
	return &LoginResult{
		User:         UserViewFrom(u),
		AuthToken:    tokens.AuthToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}
