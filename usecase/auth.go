package usecase

import (
	"context"
	"log/slog"
	"time"

	"crowdfund-app/auth"
	"crowdfund-app/entity"
	"crowdfund-app/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	refreshTokenTTL = 7 * 24 * time.Hour
	resetTokenTTL   = time.Hour
)

type Mailer interface {
	SendPasswordReset(ctx context.Context, email, plainToken string) error
}

type LogMailer struct {
	AppPublicURL string
}

func (m LogMailer) SendPasswordReset(_ context.Context, email, plainToken string) error {
	slog.Info("password_reset_email", "to", email, "url", auth.PasswordResetURL(m.AppPublicURL, plainToken))
	return nil
}

type SessionTokens struct {
	AuthToken    string
	RefreshToken string
}

type LoginResult struct {
	User         UserView
	AuthToken    string
	RefreshToken string
	Requires2FA  bool
}

type SessionInfo struct {
	Role      entity.UserRole
	AdminRole *entity.AdminRole
}

type AuthUseCase struct {
	Users         repository.UserRepo
	SellerRegs    repository.SellerRegRepo
	RefreshTokens repository.RefreshTokenRepo
	ResetToks     repository.ResetRepo
	Activity      repository.ActivityRepo
	OAuth         repository.OAuthRepo
	OAuthCache    repository.OAuthCache
	TOTPBackup    repository.TOTPBackupRepo
	Tokens        *auth.TokenService
	Mailer        Mailer
	KYCStore      repository.KYCStore
	Google        *auth.GoogleOAuth
	TOTPKey       []byte
	TOTPIssuer    string
	DB            *gorm.DB
	AppPublicURL  string
}

func (a *AuthUseCase) logActivity(ctx context.Context, userID *uuid.UUID, action entity.ActivityAction, resource, resourceID string, meta RequestMeta, extra entity.JSONMetadata) {
	LogActivity(ctx, a.Activity, userID, action, resource, resourceID, meta, extra)
}

func twoFAVerified(u *entity.User) bool {
	if !u.MustSetup2FA() {
		return true
	}
	return u.TOTPEnabled
}

func (a *AuthUseCase) issueSession(ctx context.Context, u *entity.User, verified2FA bool) (*SessionTokens, error) {
	var (
		authToken string
		err       error
		plain     string
		now       time.Time
		row       *entity.RefreshToken
	)
	// 1. JWT 発行（token_version を tv クレームに含める）
	authToken, err = a.Tokens.Issue(auth.AuthClaims{
		UserID:        u.ID,
		Role:          u.Role,
		TwoFAVerified: verified2FA,
		AdminRole:     u.AdminRole,
		TokenVersion:  u.TokenVersion,
	})
	if err != nil {
		return nil, err
	}
	// 2. Refresh Token 平文を生成
	plain, err = auth.OpaqueHex()
	if err != nil {
		return nil, err
	}
	now = time.Now().UTC()
	// 3. DB に INSERT（通常接続、WithTx 外）
	row = &entity.RefreshToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: auth.OpaqueHash(plain),
		ExpiresAt: now.Add(refreshTokenTTL),
		CreatedAt: now,
	}
	if err = a.RefreshTokens.Create(ctx, row); err != nil {
		return nil, err
	}
	return &SessionTokens{AuthToken: authToken, RefreshToken: plain}, nil
}

func (a *AuthUseCase) ValidateSession(ctx context.Context, claims auth.AuthClaims) (SessionInfo, error) {
	var (
		u   *entity.User
		err error
	)
	u, err = a.Users.FindID(ctx, claims.UserID)
	if err != nil {
		return SessionInfo{}, err
	}
	if u == nil || !u.IsActive() {
		return SessionInfo{}, ErrInactiveUser
	}
	if claims.TokenVersion != u.TokenVersion {
		return SessionInfo{}, ErrTokenRevoked
	}
	return SessionInfo{Role: u.Role, AdminRole: u.AdminRole}, nil
}
