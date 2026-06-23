package usecase

import (
	"context"
	"log/slog"
	"time"

	"crowdfund-app/auth"
	"crowdfund-app/entity"
	"crowdfund-app/repository"
	"crowdfund-app/storage"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	reTokenTTL    = 7 * 24 * time.Hour
	resetTokenTTL = time.Hour
)

type Mailer interface {
	SendPasswordReset(ctx context.Context, email, resetURL string) error
}

type LogMailer struct{}

func (LogMailer) SendPasswordReset(_ context.Context, email, resetURL string) error {
	slog.Info("password_reset_email", "to", email, "url", resetURL)
	return nil
}

type AuthMeta struct {
	IPAddress string
	UserAgent string
	RequestID string
}

type SessionTokens struct {
	AuthToken string
	ReToken   string
}

type LoginResult struct {
	User        *entity.User
	AuthToken   string
	ReToken     string
	Requires2FA bool
}

type AuthUseCase struct {
	Users        repository.UserRepo
	SellerRegs   repository.SellerRegRepo
	ReTokens     repository.ReTokenRepo
	ResetToks    repository.ResetRepo
	Activity     repository.ActivityRepo
	OAuth        repository.OAuthRepo
	OAuthCache   repository.OAuthCache
	TOTPBackup   repository.TOTPBackupRepo
	Tokens       *auth.TokenService
	Mailer       Mailer
	KYCStore     storage.KYCStore
	Google       *auth.GoogleOAuth
	TOTPKey      []byte
	TOTPIssuer   string
	DB           *gorm.DB
	AppPublicURL string
}

func (a *AuthUseCase) logActivity(ctx context.Context, userID *uuid.UUID, action entity.ActivityAction, resource, resourceID string, meta AuthMeta, extra entity.JSONMetadata) {
	var (
		res   *string
		rid   *string
		ip    *string
		ua    *string
		reqID *string
	)

	if a.Activity == nil {
		return
	}
	if resource != "" {
		res = &resource
	}
	if resourceID != "" {
		rid = &resourceID
	}
	if meta.IPAddress != "" {
		ip = &meta.IPAddress
	}
	if meta.UserAgent != "" {
		ua = &meta.UserAgent
	}
	if meta.RequestID != "" {
		reqID = &meta.RequestID
	}
	_ = a.Activity.Create(ctx, &entity.ActivityLog{
		ID:         uuid.New(),
		UserID:     userID,
		Action:     action,
		Resource:   res,
		ResourceID: rid,
		IPAddress:  ip,
		UserAgent:  ua,
		RequestID:  reqID,
		Metadata:   extra,
		CreatedAt:  time.Now().UTC(),
	})
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
		row       *entity.ReToken
	)

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
	plain, err = auth.OpaqueHex()
	if err != nil {
		return nil, err
	}
	now = time.Now().UTC()
	row = &entity.ReToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: auth.OpaqueHash(plain),
		ExpiresAt: now.Add(reTokenTTL),
		CreatedAt: now,
	}
	if err = a.ReTokens.Create(ctx, row); err != nil {
		return nil, err
	}
	return &SessionTokens{AuthToken: authToken, ReToken: plain}, nil
}

func (a *AuthUseCase) invalidateSessions(ctx context.Context, userID uuid.UUID) error {
	var err error

	if _, err = a.Users.BumpTokenVer(ctx, userID); err != nil {
		return err
	}
	return a.ReTokens.RevokeAllForUser(ctx, userID, time.Now().UTC())
}
