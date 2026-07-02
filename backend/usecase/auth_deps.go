package usecase

import (
	"context"
	"time"

	"crowdfund-app/model"
	"crowdfund-app/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// authDeps は認証系ユースケースが共有する依存とヘルパー。
type authDeps struct {
	Users         repository.IUserRepo
	SellerRegs    repository.ISellerRegRepo
	RefreshTokens repository.IRefreshTokenRepo
	ResetToks     repository.IResetRepo
	Activity      repository.IActivityRepo
	OAuth         repository.IOAuthRepo
	OAuthCache    repository.IOAuthCache
	TOTPBackup    repository.ITOTPBackupRepo
	Tokens        *TokenService
	Mailer        Mailer
	KYCStore      repository.IKYCStore
	Google        *GoogleOAuth
	TOTPKey       []byte
	TOTPIssuer    string
	pendingSecret []byte
	DB            *gorm.DB
	AppPublicURL  string
}

func (d *authDeps) logActivity(ctx context.Context, userID *uuid.UUID, action model.ActivityAction, resource, resourceID string, meta RequestMeta, extra model.JSONMetadata) {
	LogActivity(ctx, d.Activity, userID, action, resource, resourceID, meta, extra)
}

func (d *authDeps) issueSession(ctx context.Context, u model.User, verified2FA bool) (*SessionTokens, error) {
	var (
		authToken string
		err       error
		plain     string
		now       time.Time
		row       *model.RefreshToken
	)
	authToken, err = d.Tokens.Issue(AuthClaims{
		UserID:        u.ID,
		Role:          u.Role,
		TwoFAVerified: verified2FA,
		AdminRole:     u.AdminRole,
		TokenVersion:  u.TokenVersion,
	})
	if err != nil {
		return nil, err
	}
	plain, err = OpaqueHex()
	if err != nil {
		return nil, err
	}
	now = time.Now().UTC()
	row = &model.RefreshToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: OpaqueHash(plain),
		ExpiresAt: now.Add(refreshTokenTTL),
		CreatedAt: now,
	}
	if err = d.RefreshTokens.Create(ctx, row); err != nil {
		return nil, err
	}
	return &SessionTokens{AuthToken: authToken, RefreshToken: plain}, nil
}

func twoFAVerified(u model.User) bool {
	if !u.MustSetup2FA() {
		return true
	}
	return u.TOTPEnabled
}
