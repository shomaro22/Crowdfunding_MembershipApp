package usecase

import (
	"crowdfund-app/repository"
	"crowdfund-app/stripe"

	"gorm.io/gorm"
)

func NewProjectUsecase(projects repository.IProjectRepo, users repository.IUserRepo, favorites repository.IFavoriteRepo, reports repository.IReportRepo, db *gorm.DB) *ProjectUseCase {
	return &ProjectUseCase{Projects: projects, Users: users, Favorites: favorites, Reports: reports, DB: db}
}

func NewKYCAccessUsecase(kyc repository.IKYCStore, jwtSecret string) *KYCAccessUseCase {
	return &KYCAccessUseCase{KYC: kyc, SignSecret: []byte(jwtSecret)}
}

func NewConnectUsecase(users repository.IUserRepo, connect repository.IConnectRepo, stripeClient stripe.Client) *ConnectUseCase {
	return &ConnectUseCase{Users: users, Connect: connect, Stripe: stripeClient}
}

func NewSupporterUsecase(db *gorm.DB, users repository.IUserRepo, projects repository.IProjectRepo, supporters repository.ISupporterRepo, purchases repository.IPurchaseRepo, connect repository.IConnectRepo, activity repository.IActivityRepo, stripeClient stripe.Client) *SupporterUseCase {
	return &SupporterUseCase{
		DB: db, Users: users, Projects: projects, Supporters: supporters,
		Purchases: purchases, Connect: connect, Activity: activity, Stripe: stripeClient,
	}
}

func NewPurchaseUsecase(users repository.IUserRepo, projects repository.IProjectRepo, purchases repository.IPurchaseRepo) *PurchaseUseCase {
	return &PurchaseUseCase{Users: users, Projects: projects, Purchases: purchases}
}

func NewPaymentUsecase(db *gorm.DB, users repository.IUserRepo, projects repository.IProjectRepo, purchases repository.IPurchaseRepo, supporters repository.ISupporterRepo, connect repository.IConnectRepo, webhooks repository.IWebhookEventRepo, activity repository.IActivityRepo, stripeClient stripe.Client, connectSync *ConnectUseCase, supporterViewer *SupporterUseCase) IPaymentUsecase {
	return &PaymentUseCase{
		DB: db, Users: users, Projects: projects, Purchases: purchases, Supporters: supporters,
		Connect: connect, Webhooks: webhooks, Activity: activity, Stripe: stripeClient,
		ConnectSync: connectSync, SupporterViewer: supporterViewer, PlanUpgrades: supporterViewer,
	}
}

func NewAdminUsecase(db *gorm.DB, repos AdminRepos, kyc repository.IKYCStore, projectUC *ProjectUseCase, kycAccess *KYCAccessUseCase, purchaseUC *PurchaseUseCase) IAdminUsecase {
	return &AdminUseCase{
		Users: repos.Users, SellerRegs: repos.SellerRegs, Projects: repos.Projects,
		Audit: repos.Audit, ReviewDecisions: repos.ReviewDecisions, RefreshTokens: repos.RefreshTokens,
		TOTPBackup: repos.TOTPBackup, KYC: kyc, ProjectsUC: projectUC, KYCAccess: kycAccess,
		AdminStats: repos.Admin, Purchases: repos.Purchases, PurchaseViews: purchaseUC, DB: db,
	}
}

type AdminRepos struct {
	Users           repository.IUserRepo
	SellerRegs      repository.ISellerRegRepo
	Projects        repository.IProjectRepo
	Audit           repository.IAuditRepo
	ReviewDecisions repository.IReviewDecisionRepo
	RefreshTokens   repository.IRefreshTokenRepo
	TOTPBackup      repository.ITOTPBackupRepo
	Admin           repository.IAdminRepo
	Purchases       repository.IPurchaseRepo
}

func NewRateLimitUsecase(repo repository.IRateLimitRepo) IRateLimitUsecase {
	return &RateLimitUseCase{Repo: repo}
}

func NewMessageUsecase(messages repository.IMessageRepo, projects repository.IProjectRepo, supporters repository.ISupporterRepo, sellers repository.ISellerRepo, users repository.IUserRepo, activity repository.IActivityRepo) *MessageUseCase {
	return &MessageUseCase{
		Messages: messages, Projects: projects, Supporters: supporters, Sellers: sellers, Users: users, Activity: activity,
	}
}

func NewWSTokenUsecase(tokens repository.IWSTokenRepo, users repository.IUserRepo, activity repository.IActivityRepo) IWSTokenUsecase {
	return &WSTokenUseCase{Tokens: tokens, Users: users, Activity: activity}
}

func NewOperationsUsecase(activity repository.IActivityRepo, audit repository.IAuditRepo, batch repository.IBatchRepo, users repository.IUserRepo, batchUC *BatchUseCase) IOperationsUsecase {
	return &OperationsUseCase{Activity: activity, Audit: audit, Runs: batch, Users: users, Batch: batchUC}
}

func NewFavoriteUsecase(favorites repository.IFavoriteRepo, projects *ProjectUseCase, activity repository.IActivityRepo) IFavoriteUsecase {
	return &FavoriteUseCase{Favorites: favorites, Projects: projects, Activity: activity}
}

func NewReportUsecase(reports repository.IReportRepo, users repository.IUserRepo, projects repository.IProjectRepo, messages repository.IMessageRepo, activity repository.IActivityRepo, audit repository.IAuditRepo) IReportUsecase {
	return &ReportUseCase{Reports: reports, Users: users, Projects: projects, Messages: messages, Activity: activity, Audit: audit}
}

func NewMediaUsecase(store repository.IMediaStore, apiPublicURL string) IMediaUsecase {
	return &MediaUseCase{Store: store, APIPublicURL: apiPublicURL}
}

func NewSellerUsecase(projects repository.IProjectRepo, seller repository.ISellerRepo, supporters *SupporterUseCase) ISellerUsecase {
	return &SellerUseCase{Projects: projects, Seller: seller, Supporters: supporters}
}

func NewBatchUsecase(db *gorm.DB, runs repository.IBatchRepo, supporters *SupporterUseCase, wsTokens repository.IWSTokenRepo) *BatchUseCase {
	return &BatchUseCase{DB: db, Runs: runs, Supporters: supporters, WSTokens: wsTokens}
}
