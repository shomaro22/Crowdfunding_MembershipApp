package usecase

import (
	"context"
	"log/slog"
	"time"

	"crowdfund-app/auth"
	"crowdfund-app/model"
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
	Role      model.UserRole
	AdminRole *model.AdminRole
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

func (a *AuthUseCase) logActivity(ctx context.Context, userID *uuid.UUID, action model.ActivityAction, resource, resourceID string, meta RequestMeta, extra model.JSONMetadata) {
	LogActivity(ctx, a.Activity, userID, action, resource, resourceID, meta, extra)
}

func twoFAVerified(u *model.User) bool {
	if !u.MustSetup2FA() {
		return true
	}
	return u.TOTPEnabled
}
