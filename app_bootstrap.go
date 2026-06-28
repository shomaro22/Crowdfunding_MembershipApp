package main

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	"crowdfund-app/auth"
	"crowdfund-app/bootstrap"
	"crowdfund-app/controller"
	"crowdfund-app/entity"
	"crowdfund-app/middleware"
	"crowdfund-app/router"

	"github.com/labstack/echo/v4"
	labmw "github.com/labstack/echo/v4/middleware"
)

func buildApp(opts *bootstrap.Options) (*echo.Echo, string, error) {
	var (
		app             *bootstrap.App
		err             error
		cookieCfg       auth.CookieCfg
		authCtl         *controller.AuthController
		healthCtl       *controller.HealthController
		adminCtl        *controller.AdminController
		adminOpsCtl     *controller.AdminOperationsController
		kycFileCtl      *controller.KYCFileController
		projectCtl      *controller.ProjectController
		connectCtl      *controller.ConnectController
		paymentCtl      *controller.PaymentController
		supporterCtl    *controller.SupporterController
		favoriteCtl     *controller.FavoriteController
		messageCtl      *controller.MessageController
		wsTokenCtl      *controller.WSTokenController
		wsCtl           *controller.WSController
		preAuth         []echo.MiddlewareFunc
		corsRaw         string
		origins         []string
		p               string
		s               string
		protected       []echo.MiddlewareFunc
		optionalJWT     []echo.MiddlewareFunc
		sellerRead      []echo.MiddlewareFunc
		sellerWrite     []echo.MiddlewareFunc
		adminOnly       []echo.MiddlewareFunc
		rateLimiter     *middleware.RateLimiter
		e               *echo.Echo
		validateSession middleware.SessionValidator
	)

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	app, err = bootstrap.OpenApp(opts)
	if err != nil {
		return nil, "", err
	}

	corsRaw = strings.TrimSpace(os.Getenv("CORS_ALLOW_ORIGINS"))
	if corsRaw != "" {
		for _, p = range strings.Split(corsRaw, ",") {
			s = strings.TrimSpace(p)
			if s != "" {
				origins = append(origins, s)
			}
		}
	}

	cookieCfg = auth.CookieFromEnv()
	authCtl = controller.NewAuthController(app.UseCases.Auth, cookieCfg, app.Config.AppPublicURL)
	healthCtl = &controller.HealthController{SQL: app.DB, RDB: app.RDB}
	adminCtl = controller.NewAdminController(app.UseCases.Admin, app.Config.APIPublicURL)
	adminOpsCtl = controller.NewAdminOperationsController(app.UseCases.Operations)
	kycFileCtl = controller.NewKYCFileController(app.UseCases.KYCAccess)
	projectCtl = controller.NewProjectController(app.UseCases.Project)
	connectCtl = controller.NewConnectController(app.UseCases.Connect)
	if app.Stripe != nil {
		paymentCtl = controller.NewPaymentController(app.UseCases.Payment, app.UseCases.Purchase, app.Stripe.AppURL())
	}
	supporterCtl = controller.NewSupporterController(app.UseCases.Supporter)
	favoriteCtl = controller.NewFavoriteController(app.UseCases.Favorite)
	messageCtl = controller.NewMessageController(app.UseCases.Message)
	wsTokenCtl = controller.NewWSTokenController(app.UseCases.WSToken)
	wsCtl = controller.NewWSController(app.Hub, app.UseCases.Message, origins)

	validateSession = middleware.SessionValidateHTTP(app.UseCases.Auth.ValidateSession)

	if len(origins) > 0 {
		preAuth = append(preAuth, labmw.CORSWithConfig(labmw.CORSConfig{
			AllowOrigins:     origins,
			AllowCredentials: true,
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
			AllowMethods: []string{
				http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch,
				http.MethodPost, http.MethodDelete, http.MethodOptions,
			},
		}))
	}

	protected = []echo.MiddlewareFunc{
		middleware.JWTAuth(app.UseCases.Auth.ParseToken, validateSession),
	}
	optionalJWT = []echo.MiddlewareFunc{
		middleware.OptionalJWTAuth(app.UseCases.Auth.ParseToken, validateSession),
	}
	sellerRead = append(append([]echo.MiddlewareFunc{}, protected...), middleware.RequireRole(entity.UserRoleSeller))
	sellerWrite = append(append([]echo.MiddlewareFunc{}, sellerRead...), middleware.Require2FA())
	adminOnly = append(
		append([]echo.MiddlewareFunc{}, protected...),
		middleware.RequireRole(entity.UserRoleAdmin),
		middleware.Require2FA(),
	)
	rateLimiter = middleware.NewRateLimiter(app.UseCases.RateLimit)
	bootstrap.WireRateLimitActivity(app, rateLimiter)

	e = router.New(router.Config{
		PreAuthMiddleware:  preAuth,
		Health:             healthCtl,
		Auth:               authCtl,
		Projects:           projectCtl,
		Connect:            connectCtl,
		Payments:           paymentCtl,
		Supporters:         supporterCtl,
		Favorites:          favoriteCtl,
		Messages:           messageCtl,
		WSToken:            wsTokenCtl,
		WS:                 wsCtl,
		WSTokenAuth:        middleware.WSTokenAuth(app.UseCases.WSToken.Consume),
		Admin:              adminCtl,
		AdminOperations:    adminOpsCtl,
		KYCFiles:           kycFileCtl,
		RateLimit:          rateLimiter,
		Policy2FAChallenge: middleware.Policy2FAChallenge([]byte(app.Config.JWTSecret)),
		OAuthEnabled:       app.UseCases.Auth.Google != nil,
		Protected:          protected,
		OptionalJWT:        optionalJWT,
		SellerRead:         sellerRead,
		SellerWrite:        sellerWrite,
		AdminOnly:          adminOnly,
	})
	return e, app.Config.ListenAddr(), nil
}
