package bootstrap

import (
	"crowdfund-app/repository"
	"crowdfund-app/stripe"
	"crowdfund-app/usecase"

	"gorm.io/gorm"
)

// UseCases はユースケース層の依存をまとめる。
type UseCases struct {
	Auth        usecase.IAuthUsecase
	Project     usecase.IProjectUsecase
	ProjectRaw  *usecase.ProjectUseCase
	KYCAccess   usecase.IKYCAccessUsecase
	KYCRaw      *usecase.KYCAccessUseCase
	Connect     usecase.IConnectUsecase
	ConnectRaw  *usecase.ConnectUseCase
	Supporter   usecase.ISupporterUsecase
	SupporterRaw *usecase.SupporterUseCase
	Purchase    usecase.IPurchaseUsecase
	PurchaseRaw *usecase.PurchaseUseCase
	Payment     usecase.IPaymentUsecase
	Admin       usecase.IAdminUsecase
	Message     usecase.IMessageUsecase
	MessageRaw  *usecase.MessageUseCase
	WSToken     usecase.IWSTokenUsecase
	Operations  usecase.IOperationsUsecase
	Favorite    usecase.IFavoriteUsecase
	Report      usecase.IReportUsecase
	Media       usecase.IMediaUsecase
	Seller      usecase.ISellerUsecase
	RateLimit   usecase.IRateLimitUsecase
	Batch       *usecase.BatchUseCase
}

// NewUseCases はリポジトリからユースケース群を構築する。
func NewUseCases(repos Repositories, cfg AppConfig, gormDB *gorm.DB, stripeClient stripe.Client) UseCases {
	var (
		ucs UseCases
		tx  = repository.NewTxBeginner(gormDB)
	)
	ucs.Auth = usecase.NewAuthUsecase(usecase.AuthConfig{
		Users: repos.Users, SellerRegs: repos.SellerRegs, RefreshTokens: repos.RefreshTokens,
		ResetToks: repos.ResetToks, Activity: repos.Activity, OAuth: repos.OAuth,
		OAuthCache: repos.OAuthCache, TOTPBackup: repos.TOTPBackup, KYCStore: repos.KYCStore,
		JWTSecret: cfg.Security.JWTSecret, TOTPEncryptionKey: cfg.Security.TOTPEncryptionKey,
		Pending2FAKey: cfg.Security.Pending2FAKey, TOTPIssuer: cfg.Security.TOTPIssuer,
		AppPublicURL: cfg.OAuth.AppPublicURL, Tx: tx,
		Mailer: usecase.NewLogMailer(cfg.OAuth.AppPublicURL),
	})
	ucs.ProjectRaw = usecase.NewProjectUsecase(repos.Projects, repos.Users, repos.Favorites, repos.Reports, tx)
	ucs.Project = ucs.ProjectRaw
	ucs.KYCRaw = usecase.NewKYCAccessUsecase(repos.KYCStore, string(cfg.Security.KYCSigningKey))
	ucs.KYCAccess = ucs.KYCRaw
	ucs.ConnectRaw = usecase.NewConnectUsecase(repos.Users, repos.Connect, stripeClient)
	ucs.Connect = ucs.ConnectRaw
	ucs.SupporterRaw = usecase.NewSupporterUsecase(
		tx, repos.Users, repos.Projects, repos.Supporters, repos.Purchases,
		repos.Connect, repos.Activity, stripeClient,
	)
	ucs.Supporter = ucs.SupporterRaw
	ucs.PurchaseRaw = usecase.NewPurchaseUsecase(repos.Users, repos.Projects, repos.Purchases)
	ucs.Purchase = ucs.PurchaseRaw
	paymentRaw := usecase.NewPaymentUsecase(
		tx, repos.Users, repos.Projects, repos.Purchases, repos.Supporters,
		repos.Connect, repos.Webhooks, repos.Activity, stripeClient, ucs.ConnectRaw, ucs.SupporterRaw,
	)
	ucs.Payment = paymentRaw
	AttachSupportEvents(paymentRaw, NewSupportEventBus(repos.Activity))
	ucs.Batch = usecase.NewBatchUsecase(repos.Batch, ucs.SupporterRaw, repos.WSTokens)
	ucs.Admin = usecase.NewAdminUsecase(tx, usecase.AdminRepos{
		Users: repos.Users, SellerRegs: repos.SellerRegs, Projects: repos.Projects, Audit: repos.Audit,
		ReviewDecisions: repos.ReviewDecisions, RefreshTokens: repos.RefreshTokens, TOTPBackup: repos.TOTPBackup,
		Admin: repos.Admin, Purchases: repos.Purchases,
	}, repos.KYCStore, ucs.ProjectRaw, ucs.KYCRaw, ucs.PurchaseRaw)
	ucs.MessageRaw = usecase.NewMessageUsecase(
		repos.Messages, repos.Projects, repos.Supporters, repos.Seller, repos.Users, repos.Activity,
	)
	AttachMessageEvents(ucs.MessageRaw, repos.WSHub)
	ucs.Message = ucs.MessageRaw
	ucs.WSToken = usecase.NewWSTokenUsecase(repos.WSTokens, repos.Users, repos.Activity)
	ucs.Operations = usecase.NewOperationsUsecase(repos.Activity, repos.Audit, repos.Batch, repos.Users, ucs.Batch)
	ucs.Favorite = usecase.NewFavoriteUsecase(repos.Favorites, ucs.ProjectRaw, repos.Activity)
	ucs.Report = usecase.NewReportUsecase(repos.Reports, repos.Users, repos.Projects, repos.Messages, repos.Activity, repos.Audit)
	ucs.Media = usecase.NewMediaUsecase(repos.MediaStore, cfg.OAuth.APIPublicURL)
	ucs.Seller = usecase.NewSellerUsecase(repos.Projects, repos.Seller, ucs.SupporterRaw)
	ucs.RateLimit = usecase.NewRateLimitUsecase(repos.RateLimit)
	return ucs
}
