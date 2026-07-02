package bootstrap

import (
	"net/http"

	"crowdfund-app/middleware"
	"crowdfund-app/model"
	"crowdfund-app/repository"
	"crowdfund-app/router"
	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	labmw "github.com/labstack/echo/v4/middleware"
)

// MiddlewareBundle は Echo ミドルウェア群。
type MiddlewareBundle struct {
	PreAuth         []echo.MiddlewareFunc
	Protected       []echo.MiddlewareFunc
	OptionalJWT     []echo.MiddlewareFunc
	SellerRead      []echo.MiddlewareFunc
	SellerWrite     []echo.MiddlewareFunc
	AdminOnly       []echo.MiddlewareFunc
	RateLimiter     *middleware.RateLimiter
	Policy2FAChallenge middleware.RatePolicy
}

// BuildMiddlewares は認証・レート制限ミドルウェアを構築する。
func BuildMiddlewares(
	authUC usecase.IAuthUsecase,
	activity repository.IActivityRepo,
	rateLimitUC usecase.IRateLimitUsecase,
	cfg AppConfig,
	origins []string,
) MiddlewareBundle {
		var (
		bundle          MiddlewareBundle
		validateSession middleware.SessionValidator
	)
	validateSession = middleware.SessionValidateHTTP(authUC.ValidateSession)
	if len(origins) > 0 {
		bundle.PreAuth = append(bundle.PreAuth, labmw.CORSWithConfig(labmw.CORSConfig{
			AllowOrigins:     origins,
			AllowCredentials: true,
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
			AllowMethods: []string{
				http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch,
				http.MethodPost, http.MethodDelete, http.MethodOptions,
			},
		}))
	}
	bundle.Protected = []echo.MiddlewareFunc{
		middleware.JWTAuth(authUC.ParseToken, validateSession),
	}
	bundle.OptionalJWT = []echo.MiddlewareFunc{
		middleware.OptionalJWTAuth(authUC.ParseToken, validateSession),
	}
	bundle.SellerRead = make([]echo.MiddlewareFunc, len(bundle.Protected))
	copy(bundle.SellerRead, bundle.Protected)
	bundle.SellerRead = append(bundle.SellerRead, middleware.RequireRole(model.UserRoleSeller))
	bundle.SellerWrite = make([]echo.MiddlewareFunc, len(bundle.SellerRead))
	copy(bundle.SellerWrite, bundle.SellerRead)
	bundle.SellerWrite = append(bundle.SellerWrite, middleware.Require2FA())
	bundle.AdminOnly = make([]echo.MiddlewareFunc, len(bundle.Protected))
	copy(bundle.AdminOnly, bundle.Protected)
	bundle.AdminOnly = append(bundle.AdminOnly,
		middleware.RequireRole(model.UserRoleAdmin),
		middleware.Require2FA(),
	)
	bundle.RateLimiter = middleware.NewRateLimiter(rateLimitUC)
	wireRateLimitActivity(activity, bundle.RateLimiter)
	bundle.Policy2FAChallenge = middleware.Policy2FAChallenge(cfg.Security.Pending2FAKey)
	return bundle
}

func wireRateLimitActivity(activity repository.IActivityRepo, limiter *middleware.RateLimiter) {
	if limiter == nil {
		return
	}
	limiter.OnDenied = func(c echo.Context, policy middleware.RatePolicy, ruleKey string) {
		var (
			uid    *uuid.UUID
			userID uuid.UUID
			ok     bool
			meta   usecase.RequestMeta
		)
		if userID, ok = middleware.AuthUserID(c); ok {
			uid = &userID
		}
		meta = httpRequestMeta(c)
		usecase.LogRateLimitDenied(c.Request().Context(), activity, uid, policy.Name, ruleKey, meta)
	}
}

func httpRequestMeta(c echo.Context) usecase.RequestMeta {
	reqID, _ := c.Get(middleware.CtxRequestID).(string)
	return usecase.RequestMeta{
		IPAddress: c.RealIP(),
		UserAgent: c.Request().UserAgent(),
		RequestID: reqID,
	}
}

// BuildRouter はルータを構築する。
func BuildRouter(
	ctls Controllers,
	ucs UseCases,
	mw MiddlewareBundle,
	cfg AppConfig,
) *echo.Echo {
	return router.New(router.Config{
		PreAuthMiddleware:  mw.PreAuth,
		Health:             ctls.Health,
		AuthUsecase:        ucs.Auth,
		AppPublicURL:       cfg.OAuth.AppPublicURL,
		Projects:           ctls.Projects,
		Connect:            ctls.Connect,
		Seller:             ctls.Seller,
		Payments:           ctls.Payments,
		Supporters:         ctls.Supporters,
		Favorites:          ctls.Favorites,
		Reports:            ctls.Reports,
		Media:              ctls.Media,
		Messages:           ctls.Messages,
		WSToken:            ctls.WSToken,
		WS:                 ctls.WS,
		WSTokenAuth:        middleware.WSTokenAuth(ucs.WSToken.Consume),
		Admin:              ctls.Admin,
		AdminOperations:    ctls.AdminOperations,
		KYCFiles:           ctls.KYCFiles,
		RateLimit:          mw.RateLimiter,
		Policy2FAChallenge: mw.Policy2FAChallenge,
		OAuthEnabled:       ucs.Auth.OAuthEnabled(),
		Protected:          mw.Protected,
		OptionalJWT:        mw.OptionalJWT,
		SellerRead:         mw.SellerRead,
		SellerWrite:        mw.SellerWrite,
		AdminOnly:          mw.AdminOnly,
	})
}
