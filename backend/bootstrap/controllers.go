package bootstrap

import (
	"crowdfund-app/controller"
	"crowdfund-app/stripe"
	"crowdfund-app/usecase"
	"crowdfund-app/wshub"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Controllers は HTTP コントローラ群。
type Controllers struct {
	Health          controller.IHealthController
	Admin           controller.IAdminController
	AdminOperations controller.IAdminOperationsController
	KYCFiles        controller.IKYCFileController
	Projects        controller.IProjectController
	Connect         controller.IConnectController
	Payments        controller.IPaymentController
	Supporters      controller.ISupporterController
	Favorites       controller.IFavoriteController
	Reports         controller.IReportController
	Media           controller.IMediaController
	Seller          controller.ISellerController
	Messages        controller.IMessageController
	WSToken         controller.IWSTokenController
	WS              controller.IWSController
}

// NewControllers はユースケースからコントローラを構築する。
func NewControllers(
	ucs UseCases,
	kyc usecase.IKYCAccessUsecase,
	gormDB *gorm.DB,
	rdb *redis.Client,
	hub *wshub.ConnectionManager,
	cfg AppConfig,
	stripeClient stripe.Client,
	corsOrigins []string,
) Controllers {
	var (
		ctls Controllers
	)
	ctls.Health = controller.NewHealthController(gormDB, rdb)
	ctls.Admin = controller.NewAdminController(ucs.Admin, cfg.OAuth.APIPublicURL)
	ctls.AdminOperations = controller.NewAdminOperationsController(ucs.Operations)
	ctls.KYCFiles = controller.NewKYCFileController(kyc)
	ctls.Projects = controller.NewProjectController(ucs.Project)
	ctls.Connect = controller.NewConnectController(ucs.Connect)
	if stripeClient != nil {
		ctls.Payments = controller.NewPaymentController(ucs.Payment, ucs.Purchase, stripeClient.AppURL())
	}
	ctls.Supporters = controller.NewSupporterController(ucs.Supporter)
	ctls.Favorites = controller.NewFavoriteController(ucs.Favorite)
	ctls.Reports = controller.NewReportController(ucs.Report)
	ctls.Media = controller.NewMediaController(ucs.Media)
	ctls.Seller = controller.NewSellerController(ucs.Seller)
	ctls.Messages = controller.NewMessageController(ucs.Message)
	ctls.WSToken = controller.NewWSTokenController(ucs.WSToken)
	ctls.WS = controller.NewWSController(hub, ucs.Message, corsOrigins)
	return ctls
}
