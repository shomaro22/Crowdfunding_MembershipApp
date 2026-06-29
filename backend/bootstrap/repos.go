package bootstrap

import (
	"crowdfund-app/repository"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Repositories struct {
	Users           repository.IUserRepo
	SellerRegs      repository.ISellerRegRepo
	RefreshTokens   repository.IRefreshTokenRepo
	ResetToks       repository.IResetRepo
	Activity        repository.IActivityRepo
	Audit           repository.IAuditRepo
	ReviewDecisions repository.IReviewDecisionRepo
	Projects        repository.IProjectRepo
	Connect         repository.IConnectRepo
	Webhooks        repository.IWebhookEventRepo
	Supporters      repository.ISupporterRepo
	Purchases       repository.IPurchaseRepo
	OAuth           repository.IOAuthRepo
	TOTPBackup      repository.ITOTPBackupRepo
	Batch           repository.IBatchRepo
	Messages        repository.IMessageRepo
	WSTokens        repository.IWSTokenRepo
	Favorites       repository.IFavoriteRepo
	Reports         repository.IReportRepo
	Seller          repository.ISellerRepo
	Admin           repository.IAdminRepo
}

type RedisRepositories struct {
	OAuthCache repository.IOAuthCache
	RateLimit  repository.IRateLimitRepo
}

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{
		Users:           repository.NewUserRepo(db),
		SellerRegs:      repository.NewSellerRegRepo(db),
		RefreshTokens:   repository.NewRefreshTokenRepo(db),
		ResetToks:       repository.NewResetRepo(db),
		Activity:        repository.NewActivityRepo(db),
		Audit:           repository.NewAuditRepo(db),
		ReviewDecisions: repository.NewReviewDecisionRepo(db),
		Projects:        repository.NewProjectRepo(db),
		Connect:         repository.NewConnectRepo(db),
		Webhooks:        repository.NewWebhookEventRepo(db),
		Supporters:      repository.NewSupporterRepo(db),
		Purchases:       repository.NewPurchaseRepo(db),
		OAuth:           repository.NewOAuthRepo(db),
		TOTPBackup:      repository.NewTOTPBackupRepo(db),
		Batch:           repository.NewBatchRepo(db),
		Messages:        repository.NewMessageRepo(db),
		WSTokens:        repository.NewWSTokenRepo(db),
		Favorites:       repository.NewFavoriteRepo(db),
		Reports:         repository.NewReportRepo(db),
		Seller:          repository.NewSellerRepo(db),
		Admin:           repository.NewAdminRepo(db),
	}
}

func NewRedisRepositories(rdb *redis.Client) RedisRepositories {
	return RedisRepositories{
		OAuthCache: repository.NewOAuthCache(rdb),
		RateLimit:  repository.NewRateLimitRepo(rdb),
	}
}
