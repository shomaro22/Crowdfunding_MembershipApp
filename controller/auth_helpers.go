package controller

import (
	"errors"
	"net/http"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/middleware"
	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func CurrentUserID(c echo.Context) (uuid.UUID, error) {
	uid, ok := middleware.AuthUserID(c)
	if !ok {
		return uuid.Nil, echo.NewHTTPError(http.StatusUnauthorized, apiError("unauthorized", "UNAUTHORIZED"))
	}
	return uid, nil
}

func authMeta(c echo.Context) usecase.AuthMeta {
	reqID, _ := c.Get(middleware.CtxRequestID).(string)
	return usecase.AuthMeta{
		IPAddress: c.RealIP(),
		UserAgent: c.Request().UserAgent(),
		RequestID: reqID,
	}
}

func userToJSON(u *entity.User) userJSON {
	if u == nil {
		return userJSON{}
	}
	return userJSON{
		ID:        u.ID.String(),
		Email:     u.Email,
		Name:      u.Name,
		Role:      string(u.Role),
		IsActive:  u.IsActive(),
		AvatarURL: u.AvatarURL,
		CreatedAt: u.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: u.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

type userJSON struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Name      *string `json:"name"`
	Role      string  `json:"role"`
	IsActive  bool    `json:"is_active"`
	AvatarURL *string `json:"avatar_url"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

func apiError(msg, code string) map[string]string {
	return map[string]string{"error": msg, "code": code}
}

func mapAuthError(err error) *echo.HTTPError {
	switch {
	case errors.Is(err, usecase.ErrInvalidCredentials):
		return echo.NewHTTPError(http.StatusUnauthorized, apiError("invalid credentials", "INVALID_CREDENTIALS"))
	case errors.Is(err, usecase.ErrInactiveUser):
		return echo.NewHTTPError(http.StatusForbidden, apiError("user inactive", "USER_INACTIVE"))
	case errors.Is(err, usecase.ErrEmailInUse):
		return echo.NewHTTPError(http.StatusConflict, apiError("email already in use", "EMAIL_IN_USE"))
	case errors.Is(err, usecase.ErrWeakPassword):
		return echo.NewHTTPError(http.StatusBadRequest, apiError("weak password", "WEAK_PASSWORD"))
	case errors.Is(err, usecase.ErrInvalidEmail):
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid email", "INVALID_EMAIL"))
	case errors.Is(err, usecase.ErrInvalidToken):
		return echo.NewHTTPError(http.StatusUnauthorized, apiError("invalid token", "INVALID_TOKEN"))
	case errors.Is(err, usecase.ErrNotFound):
		return echo.NewHTTPError(http.StatusNotFound, apiError("not found", "NOT_FOUND"))
	case errors.Is(err, usecase.ErrInvalid2FACode):
		return echo.NewHTTPError(http.StatusUnauthorized, apiError("invalid 2fa code", "INVALID_2FA_CODE"))
	case errors.Is(err, usecase.Err2FAAlreadyEnabled):
		return echo.NewHTTPError(http.StatusConflict, apiError("2fa already enabled", "2FA_ALREADY_ENABLED"))
	case errors.Is(err, usecase.ErrOAuthUnavailable):
		return echo.NewHTTPError(http.StatusServiceUnavailable, apiError("oauth unavailable", "OAUTH_UNAVAILABLE"))
	case errors.Is(err, usecase.ErrOAuthForbidden):
		return echo.NewHTTPError(http.StatusForbidden, apiError("oauth forbidden", "OAUTH_FORBIDDEN"))
	default:
		return echo.NewHTTPError(http.StatusInternalServerError, apiError("internal error", "INTERNAL_ERROR"))
	}
}
