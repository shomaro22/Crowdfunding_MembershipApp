package controller

import (
	"crowdfund-app/auth"
)

type AuthController struct {
	Auth         AuthUseCase
	CookieCfg    auth.CookieCfg
	AppPublicURL string
}

func NewAuthController(authUC AuthUseCase, cookieCfg auth.CookieCfg, appPublicURL string) *AuthController {
	return &AuthController{Auth: authUC, CookieCfg: cookieCfg, AppPublicURL: appPublicURL}
}
