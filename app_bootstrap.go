package main

import (
	"errors"
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
	"crowdfund-app/usecase"

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
		purchaseCtl     *controller.PurchaseController
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
		validateSession func(echo.Context, auth.AuthClaims) error
	)

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	app, err = bootstrap.OpenApp(opts)
	if err != nil {
		return nil, "", err
	}

	cookieCfg = auth.CookieFromEnv()
	authCtl = controller.NewAuthController(app.UseCases.Auth, cookieCfg)
	healthCtl = &controller.HealthController{SQL: app.DB, RDB: app.RDB}
	adminCtl = controller.NewAdminController(app.UseCases.Admin)
	kycFileCtl = controller.NewKYCFileController(app.UseCases.KYCAccess)
	projectCtl = controller.NewProjectController(app.UseCases.Project)
	connectCtl = controller.NewConnectController(app.UseCases.Connect)
	paymentCtl = controller.NewPaymentController(app.UseCases.Payment)
	supporterCtl = controller.NewSupporterController(app.UseCases.Supporter)
	purchaseCtl = controller.NewPurchaseController(app.UseCases.Purchase)

	validateSession = func(c echo.Context, claims auth.AuthClaims) error {
		var (
			info usecase.SessionInfo
			err  error
		)
		info, err = app.UseCases.Auth.ValidateSession(c.Request().Context(), claims)
		if err != nil {
			switch {
			case errors.Is(err, usecase.ErrInactiveUser):
				return echo.NewHTTPError(http.StatusForbidden, map[string]string{"error": "user inactive", "code": "USER_INACTIVE"})
			case errors.Is(err, usecase.ErrTokenRevoked):
				return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"error": "token revoked", "code": "TOKEN_REVOKED"})
			default:
				return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "internal error", "code": "INTERNAL_ERROR"})
			}
		}
		c.Set(middleware.CtxUserRole, string(info.Role))
		middleware.SetAdminRoleFromSession(c, info.Role, info.AdminRole)
		return nil
	}

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
		Purchases:          purchaseCtl,
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
