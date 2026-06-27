package bootstrap

import (
	"crowdfund-app/repository"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Repositories struct {
	Users           repository.UserRepo
	SellerRegs      repository.SellerRegRepo
	RefreshTokens   repository.RefreshTokenRepo
	ResetToks       repository.ResetRepo
	Activity        repository.ActivityRepo
	Audit           repository.AuditRepo
	ReviewDecisions repository.ReviewDecisionRepo
	Projects        repository.ProjectRepo
	Connect         repository.ConnectRepo
	Webhooks        repository.WebhookEventRepo
	Supporters      repository.SupporterRepo
	Purchases       repository.PurchaseRepo
	OAuth           repository.OAuthRepo
	TOTPBackup      repository.TOTPBackupRepo
	Batch           repository.BatchRepo
	Messages        repository.MessageRepo
	WSTokens        repository.WSTokenRepo
}

type RedisRepositories struct {
	OAuthCache repository.OAuthCache
	RateLimit  repository.RateLimitRepo
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
	}
}

func NewRedisRepositories(rdb *redis.Client) RedisRepositories {
	return RedisRepositories{
		OAuthCache: repository.NewOAuthCache(rdb),
		RateLimit:  repository.NewRateLimitRepo(rdb),
	}
}
