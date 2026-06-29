package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"crowdfund-app/auth"
	"crowdfund-app/model"

	"github.com/google/uuid"
)

const backupCodeCount = 10

type TOTPSetupResult struct {
	Secret     string
	OTPAuthURL string
}

type TOTPVerifyResult struct {
	BackupCodes []string
}

func (a *AuthUseCase) Setup2FA(ctx context.Context, userID uuid.UUID) (*TOTPSetupResult, error) {
	var (
		u      model.User
		err    error
		issuer string
		setup  *auth.TOTPSetup
		enc    string
	)

	u, err = a.Users.GetID(ctx, userID)
	if err != nil {
		return nil, mapRepoNotFound(err)
	}
	if u.TOTPEnabled {
		return nil, Err2FAAlreadyEnabled
	}
	issuer = a.TOTPIssuer
	if issuer == "" {
		issuer = "Crowdfund"
	}
	setup, err = auth.NewTOTPSetup(issuer, u.Email)
	if err != nil {
		return nil, err
	}
	enc, err = auth.EncryptSecret(a.TOTPKey, setup.Secret)
	if err != nil {
		return nil, err
	}
	u.TOTPSecret = &enc
	u.UpdatedAt = time.Now().UTC()
	if err = a.Users.Update(ctx, &u); err != nil {
		return nil, err
	}
	return &TOTPSetupResult{Secret: setup.Secret, OTPAuthURL: setup.OTPAuthURL}, nil
}

func (a *AuthUseCase) Verify2FASetup(ctx context.Context, userID uuid.UUID, code string) (*TOTPVerifyResult, error) {
	var (
		u      model.User
		err    error
		secret string
		now    time.Time
		codes  []string
	)

	u, err = a.Users.GetID(ctx, userID)
	if err != nil {
		return nil, mapRepoNotFound(err)
	}
	if u.TOTPEnabled {
		return nil, Err2FAAlreadyEnabled
	}
	secret, err = a.decryptTOTP(u)
	if err != nil {
		return nil, Err2FANotConfigured
	}
	if !auth.VerifyTOTP(secret, code) {
		return nil, ErrInvalid2FACode
	}
	now = time.Now().UTC()
	u.TOTPEnabled = true
	u.UpdatedAt = now
	if err = a.Users.Update(ctx, &u); err != nil {
		return nil, err
	}
	codes, err = a.genBackupCodes(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	if err = InvalidateUserSessions(ctx, a.DB, a.Users, a.RefreshTokens, u.ID); err != nil {
		return nil, err
	}
	return &TOTPVerifyResult{BackupCodes: codes}, nil
}

func (a *AuthUseCase) Challenge2FA(ctx context.Context, pendingToken, code string, meta RequestMeta) (*LoginResult, error) {
	var (
		uid    uuid.UUID
		err    error
		u      model.User
		tokens *SessionTokens
	)

	uid, err = auth.VerifyPending2FA(a.TOTPKey, pendingToken, time.Now().UTC())
	if err != nil {
		return nil, ErrInvalidToken
	}
	u, err = a.Users.GetID(ctx, uid)
	if err != nil {
		return nil, mapRepoNotFoundAs(err, ErrInvalidToken)
	}
	if !u.CanLogin() {
		return nil, ErrInvalidToken
	}
	if !u.TOTPEnabled {
		return nil, Err2FANotConfigured
	}
	if !a.verifyTOTPOrBackup(ctx, u, code) {
		a.logActivity(ctx, &u.ID, model.UserLoginFail, "user", u.ID.String(), meta, model.StepMeta("2fa_fail"))
		return nil, ErrInvalid2FACode
	}
	tokens, err = a.issueSession(ctx, u, true)
	if err != nil {
		return nil, err
	}
	a.logActivity(ctx, &u.ID, model.UserLogin, "user", u.ID.String(), meta, model.StepMeta("2fa_ok"))
	return &LoginResult{
		User:         UserViewFrom(u),
		AuthToken:    tokens.AuthToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

func (a *AuthUseCase) verifyTOTPOrBackup(ctx context.Context, u model.User, code string) bool {
	var (
		secret string
		err    error
		norm   string
		rows   []model.TOTPBackupCode
		hash   string
		now    time.Time
		i      int
		row    model.TOTPBackupCode
	)

	secret, err = a.decryptTOTP(u)
	if err == nil && auth.VerifyTOTP(secret, code) {
		return true
	}
	if a.TOTPBackup == nil {
		return false
	}
	norm = strings.ToUpper(strings.TrimSpace(code))
	if norm == "" {
		return false
	}
	rows, err = a.TOTPBackup.ListUnused(ctx, u.ID)
	if err != nil {
		return false
	}
	hash = auth.OpaqueHash(norm)
	now = time.Now().UTC()
	for i = range rows {
		row = rows[i]
		if row.CodeHash == hash {
			_ = a.TOTPBackup.MarkUsed(ctx, row.ID, now)
			return true
		}
	}
	return false
}

func (a *AuthUseCase) decryptTOTP(u model.User) (string, error) {
	if u.TOTPSecret == nil || strings.TrimSpace(*u.TOTPSecret) == "" {
		return "", Err2FANotConfigured
	}
	return auth.DecryptSecret(a.TOTPKey, *u.TOTPSecret)
}

func (a *AuthUseCase) genBackupCodes(ctx context.Context, userID uuid.UUID) ([]string, error) {
	var (
		now   time.Time
		plain []string
		rows  []model.TOTPBackupCode
		i     int
		code  string
		err   error
	)

	if a.TOTPBackup == nil {
		return nil, nil
	}
	_ = a.TOTPBackup.DeleteUser(ctx, userID)
	now = time.Now().UTC()
	plain = make([]string, 0, backupCodeCount)
	rows = make([]model.TOTPBackupCode, 0, backupCodeCount)
	for i = 0; i < backupCodeCount; i++ {
		code, err = randBackupCode()
		if err != nil {
			return nil, err
		}
		plain = append(plain, code)
		rows = append(rows, model.TOTPBackupCode{
			ID:        uuid.New(),
			UserID:    userID,
			CodeHash:  auth.OpaqueHash(code),
			CreatedAt: now,
		})
	}
	if err = a.TOTPBackup.CreateBatch(ctx, rows); err != nil {
		return nil, err
	}
	return plain, nil
}

func randBackupCode() (string, error) {
	var (
		b   []byte
		err error
	)

	b = make([]byte, 4)
	if _, err = rand.Read(b); err != nil {
		return "", err
	}
	return strings.ToUpper(hex.EncodeToString(b)), nil
}
