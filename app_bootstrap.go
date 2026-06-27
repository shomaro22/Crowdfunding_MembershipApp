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
		kycFileCtl      *controller.KYCFileController
		projectCtl      *controller.ProjectController
		connectCtl      *controller.ConnectController
		paymentCtl      *controller.PaymentController
		supporterCtl    *controller.SupporterController
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

	cookieCfg = auth.CookieFromEnv()
	authCtl = controller.NewAuthController(app.UseCases.Auth, cookieCfg, app.Config.AppPublicURL)
	healthCtl = &controller.HealthController{SQL: app.DB, RDB: app.RDB}
	adminCtl = controller.NewAdminController(app.UseCases.Admin, app.Config.APIPublicURL)
	kycFileCtl = controller.NewKYCFileController(app.UseCases.KYCAccess)
	projectCtl = controller.NewProjectController(app.UseCases.Project)
	connectCtl = controller.NewConnectController(app.UseCases.Connect)
	paymentCtl = controller.NewPaymentController(app.UseCases.Payment, app.UseCases.Purchase, app.Stripe.AppURL())
	supporterCtl = controller.NewSupporterController(app.UseCases.Supporter)

	validateSession = middleware.SessionValidateHTTP(app.UseCases.Auth.ValidateSession)

	corsRaw = strings.TrimSpace(os.Getenv("CORS_ALLOW_ORIGINS"))
	if corsRaw != "" {
		for _, p = range strings.Split(corsRaw, ",") {
			s = strings.TrimSpace(p)
			if s != "" {
				origins = append(origins, s)
			}
		}
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

	e = router.New(router.Config{
		PreAuthMiddleware:  preAuth,
		Health:             healthCtl,
		Auth:               authCtl,
		Projects:           projectCtl,
		Connect:            connectCtl,
		Payments:           paymentCtl,
		Supporters:         supporterCtl,
		Admin:              adminCtl,
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
