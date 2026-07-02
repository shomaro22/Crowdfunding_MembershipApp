package bootstrap

import (
	"crowdfund-app/repository"
	"crowdfund-app/wshub"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Repositories は永続化層の依存をまとめる。
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
	OAuthCache      repository.IOAuthCache
	RateLimit       repository.IRateLimitRepo
	KYCStore        *repository.LocalKYCStore
	MediaStore      *repository.LocalMediaStore
	WSHub           *wshub.ConnectionManager
}

// NewRepositories は DB / Redis からリポジトリ群を構築する。
func NewRepositories(gormDB *gorm.DB, rdb *redis.Client, cfg AppConfig) Repositories {
	var (
		repos Repositories
	)
	repos.Users = repository.NewUserRepo(gormDB)
	repos.SellerRegs = repository.NewSellerRegRepo(gormDB)
	repos.RefreshTokens = repository.NewRefreshTokenRepo(gormDB)
	repos.ResetToks = repository.NewResetRepo(gormDB)
	repos.Activity = repository.NewActivityRepo(gormDB)
	repos.Audit = repository.NewAuditRepo(gormDB)
	repos.ReviewDecisions = repository.NewReviewDecisionRepo(gormDB)
	repos.Projects = repository.NewProjectRepo(gormDB)
	repos.Connect = repository.NewConnectRepo(gormDB)
	repos.Webhooks = repository.NewWebhookEventRepo(gormDB)
	repos.Supporters = repository.NewSupporterRepo(gormDB)
	repos.Purchases = repository.NewPurchaseRepo(gormDB)
	repos.OAuth = repository.NewOAuthRepo(gormDB)
	repos.TOTPBackup = repository.NewTOTPBackupRepo(gormDB)
	repos.Batch = repository.NewBatchRepo(gormDB)
	repos.Messages = repository.NewMessageRepo(gormDB)
	repos.WSTokens = repository.NewWSTokenRepo(gormDB)
	repos.Favorites = repository.NewFavoriteRepo(gormDB)
	repos.Reports = repository.NewReportRepo(gormDB)
	repos.Seller = repository.NewSellerRepo(gormDB)
	repos.Admin = repository.NewAdminRepo(gormDB)
	repos.OAuthCache = repository.NewOAuthCache(rdb)
	repos.RateLimit = repository.NewRateLimitRepo(rdb)
	repos.KYCStore = repository.NewLocalKYC(cfg.Storage.PrivateRoot)
	repos.MediaStore = repository.NewLocalMedia(cfg.Storage.PublicRoot)
	repos.WSHub = wshub.NewConnectionManager()
	return repos
}
