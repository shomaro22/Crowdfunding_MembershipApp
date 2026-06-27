package controller

import (
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"crowdfund-app/auth"
	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type AuthController struct {
	Auth         AuthUseCase
	CookieCfg    auth.CookieCfg
	AppPublicURL string
}

func NewAuthController(authUC AuthUseCase, cookieCfg auth.CookieCfg, appPublicURL string) *AuthController {
	return &AuthController{Auth: authUC, CookieCfg: cookieCfg, AppPublicURL: appPublicURL}
}

type registerRequest struct {
	Email    string  `json:"email"`
	Password string  `json:"password"`
	Name     *string `json:"name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type changePwRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
	TotpCode        string `json:"totp_code"`
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

type resetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (h *AuthController) Register(c echo.Context) error {
	var (
		body registerRequest
		u    usecase.UserView
		err  error
	)
	if err = c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
	u, err = h.Auth.RegisterBuyer(c.Request().Context(), body.Email, body.Password, body.Name, authMeta(c))
	if err != nil {
		return mapAuthError(err)
	}
	return c.JSON(http.StatusCreated, registerResponse{
		User:    userToJSON(u),
		Message: "Registration pending admin approval",
	})
}

func (h *AuthController) Login(c echo.Context) error {
	var (
		body loginRequest
		res  *usecase.LoginResult
		err  error
	)
	if err = c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
	res, err = h.Auth.Login(c.Request().Context(), body.Email, body.Password, authMeta(c))
	if err != nil {
		return mapAuthError(err)
	}
	if res.Requires2FA {
		auth.SetPending2FA(c, h.CookieCfg, h.Auth.SigningKey(), res.User.ID)
		return c.JSON(http.StatusOK, login2FAResponse{
			Requires2FA: true,
			User:        userToJSON(res.User),
		})
	}
	auth.SetRefreshCookie(c, h.CookieCfg, res.RefreshToken)
	return c.JSON(http.StatusOK, authTokenResponse{
		AuthToken: res.AuthToken,
		User:      userToJSON(res.User),
	})
}

func (h *AuthController) Refresh(c echo.Context) error {
	var (
		plain        string
		authToken    string
		refreshToken string
		err          error
	)
	plain = auth.RefreshTokenFrom(c, h.CookieCfg)
	authToken, refreshToken, err = h.Auth.Refresh(c.Request().Context(), plain, authMeta(c))
	if err != nil {
		return mapAuthError(err)
	}
	auth.SetRefreshCookie(c, h.CookieCfg, refreshToken)
	return c.JSON(http.StatusOK, map[string]string{"auth_token": authToken})
}

func (h *AuthController) Logout(c echo.Context) error {
	var (
		plain string
		err   error
	)
	plain = auth.RefreshTokenFrom(c, h.CookieCfg)
	if err = h.Auth.Logout(c.Request().Context(), plain, authMeta(c)); err != nil {
		return mapAuthError(err)
	}
	auth.ClearRefreshCookie(c, h.CookieCfg)
	return c.NoContent(http.StatusOK)
}

func (h *AuthController) ForgotPassword(c echo.Context) error {
	var (
		body forgotPasswordRequest
		err  error
	)
	if err = c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
	_ = h.Auth.ForgotPassword(c.Request().Context(), body.Email)
	return c.NoContent(http.StatusOK)
}

func (h *AuthController) ResetPassword(c echo.Context) error {
	var (
		body resetPasswordRequest
		err  error
	)
	if err = c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
	if err = h.Auth.ResetPassword(c.Request().Context(), body.Token, body.Password); err != nil {
		return mapAuthError(err)
	}
	return c.NoContent(http.StatusOK)
}

func (h *AuthController) Me(c echo.Context) error {
	var (
		uid uuid.UUID
		u   usecase.UserView
		err error
	)
	uid, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	u, err = h.Auth.GetUser(c.Request().Context(), uid)
	if err != nil {
		return mapAuthError(err)
	}
	return c.JSON(http.StatusOK, map[string]userJSON{"user": userToJSON(u)})
}

func (h *AuthController) ChangePassword(c echo.Context) error {
	var (
		uid  uuid.UUID
		body changePwRequest
		err  error
	)
	uid, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	if err = c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
	if err = h.Auth.ChangePassword(c.Request().Context(), uid, body.CurrentPassword, body.NewPassword, body.TotpCode); err != nil {
		return mapAuthError(err)
	}
	return c.NoContent(http.StatusOK)
}

func (h *AuthController) CheckEmail(c echo.Context) error {
	var (
		email     string
		available bool
		err       error
	)
	email = c.QueryParam("email")
	available, err = h.Auth.EmailFree(c.Request().Context(), email)
	if err != nil {
		return mapAuthError(err)
	}
	return c.JSON(http.StatusOK, map[string]bool{"available": available})
}

func (h *AuthController) Setup2FA(c echo.Context) error {
	var (
		uid uuid.UUID
		res *usecase.TOTPSetupResult
		err error
	)
	uid, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	res, err = h.Auth.Setup2FA(c.Request().Context(), uid)
	if err != nil {
		return mapAuthError(err)
	}
	return c.JSON(http.StatusOK, map[string]string{
		"secret":      res.Secret,
		"otpauth_url": res.OTPAuthURL,
	})
}

func (h *AuthController) Verify2FA(c echo.Context) error {
	var (
		uid  uuid.UUID
		body struct {
			Code string `json:"code"`
		}
		res *usecase.TOTPVerifyResult
		err error
	)
	uid, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	if err = c.Bind(&body); err != nil || body.Code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("code required", "BAD_REQUEST"))
	}
	res, err = h.Auth.Verify2FASetup(c.Request().Context(), uid, body.Code)
	if err != nil {
		return mapAuthError(err)
	}
	return c.JSON(http.StatusOK, verify2FAResponse{
		Enabled:     true,
		BackupCodes: res.BackupCodes,
	})
}

func (h *AuthController) Challenge2FA(c echo.Context) error {
	var (
		pending string
		body    struct {
			Code string `json:"code"`
		}
		res *usecase.LoginResult
		err error
	)
	pending = auth.Pending2FAFrom(c)
	if pending == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, apiError("2fa session expired", "UNAUTHORIZED"))
	}
	if err = c.Bind(&body); err != nil || body.Code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("code required", "BAD_REQUEST"))
	}
	res, err = h.Auth.Challenge2FA(c.Request().Context(), pending, body.Code, authMeta(c))
	if err != nil {
		return mapAuthError(err)
	}
	auth.ClearPending2FA(c, h.CookieCfg)
	auth.SetRefreshCookie(c, h.CookieCfg, res.RefreshToken)
	return c.JSON(http.StatusOK, authTokenResponse{
		AuthToken: res.AuthToken,
		User:      userToJSON(res.User),
	})
}

func (h *AuthController) GoogleStart(c echo.Context) error {
	var (
		from    string
		authURL string
		err     error
	)
	from = c.QueryParam("from")
	authURL, err = h.Auth.GoogleAuthURL(from)
	if err != nil {
		return mapAuthError(err)
	}
	return c.Redirect(http.StatusFound, authURL)
}

func (h *AuthController) GoogleCallback(c echo.Context) error {
	var (
		errMsg string
		code   string
		state  string
		res    *usecase.OAuthCBOut
		err    error
	)
	errMsg = c.QueryParam("error")
	if errMsg != "" {
		return c.Redirect(http.StatusFound, oauthCallbackURL(h.AppPublicURL, usecase.OAuthRedirectView{Error: errMsg}))
	}
	code = c.QueryParam("code")
	state = c.QueryParam("state")
	res, err = h.Auth.GoogleCallback(c.Request().Context(), code, state, authMeta(c))
	if err != nil {
		return c.Redirect(http.StatusFound, oauthCallbackURL(h.AppPublicURL, usecase.OAuthRedirectView{Error: oauthErrCode(err)}))
	}
	auth.SetRefreshCookie(c, h.CookieCfg, res.RefreshToken)
	return c.Redirect(http.StatusFound, oauthCallbackURL(h.AppPublicURL, res.Redirect))
}

func (h *AuthController) OAuthExchange(c echo.Context) error {
	var (
		body struct {
			Code string `json:"code"`
		}
		res *usecase.LoginResult
		err error
	)
	if err = c.Bind(&body); err != nil || body.Code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("code required", "BAD_REQUEST"))
	}
	res, err = h.Auth.ExchangeOAuthCode(c.Request().Context(), body.Code, authMeta(c))
	if err != nil {
		return mapAuthError(err)
	}
	return c.JSON(http.StatusOK, authTokenResponse{
		AuthToken: res.AuthToken,
		User:      userToJSON(res.User),
	})
}

func oauthErrCode(err error) string {
	switch {
	case errors.Is(err, usecase.ErrInactiveUser):
		return "user_inactive"
	case errors.Is(err, usecase.ErrOAuthForbidden):
		return "oauth_forbidden"
	default:
		return "oauth_failed"
	}
}

type sellerRegPayload struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	FullName     string `json:"full_name"`
	BirthDate    string `json:"birth_date"`
	Address      string `json:"address"`
	PhotoPresent bool   `json:"photo_present"`
}

type sellerRegJSON struct {
	ID           string  `json:"id"`
	UserID       string  `json:"user_id"`
	FullName     string  `json:"full_name"`
	BirthDate    string  `json:"birth_date"`
	Address      string  `json:"address"`
	PhotoPresent bool    `json:"photo_present"`
	IDDocType    string  `json:"identity_document_type"`
	HasIDFront   bool    `json:"has_identity_document_front"`
	HasIDBack    bool    `json:"has_identity_document_back"`
	Status       string  `json:"status"`
	RejectReason *string `json:"rejection_reason"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

func sellerRegToJSON(r usecase.SellerRegItemView) sellerRegJSON {
	return sellerRegJSON{
		ID:           r.ID.String(),
		UserID:       r.UserID.String(),
		FullName:     r.FullName,
		BirthDate:    r.BirthDate.Format("2006-01-02"),
		Address:      r.Address,
		PhotoPresent: r.PhotoPresent,
		IDDocType:    "DRIVERS_LICENSE",
		HasIDFront:   r.HasIDFront,
		HasIDBack:    r.HasIDBack,
		Status:       string(r.Status),
		RejectReason: r.RejectReason,
		CreatedAt:    r.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    r.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func (h *AuthController) RegisterSeller(c echo.Context) error {
	var (
		regRaw    string
		payload   sellerRegPayload
		frontFile *multipart.FileHeader
		backFile  *multipart.FileHeader
		frontImg  usecase.KYCUpload
		backImg   usecase.KYCUpload
		birthDate time.Time
		res       *usecase.SellerRegOut
		err       error
	)
	regRaw = strings.TrimSpace(c.FormValue("registration"))
	if regRaw == "" {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("registration field required", "BAD_REQUEST"))
	}
	if err = json.Unmarshal([]byte(regRaw), &payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid registration json", "BAD_REQUEST"))
	}
	frontFile, err = c.FormFile("identity_document_front")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("identity_document_front required", "BAD_REQUEST"))
	}
	backFile, err = c.FormFile("identity_document_back")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("identity_document_back required", "BAD_REQUEST"))
	}
	frontImg, err = readKYCFile(frontFile)
	if err != nil {
		return mapUseCaseError(err)
	}
	backImg, err = readKYCFile(backFile)
	if err != nil {
		return mapUseCaseError(err)
	}
	birthDate, err = time.Parse("2006-01-02", strings.TrimSpace(payload.BirthDate))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid birth_date", "BAD_REQUEST"))
	}
	res, err = h.Auth.RegisterSeller(c.Request().Context(), usecase.SellerRegIn{
		Email:        payload.Email,
		Password:     payload.Password,
		FullName:     payload.FullName,
		BirthDate:    birthDate,
		Address:      payload.Address,
		PhotoPresent: payload.PhotoPresent,
		Front:        frontImg,
		Back:         backImg,
	}, authMeta(c))
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusCreated, sellerCreateJSON{
		User:         userToJSON(res.User),
		Registration: sellerRegToJSON(res.Registration),
	})
}

func readKYCFile(fh *multipart.FileHeader) (usecase.KYCUpload, error) {
	var (
		src         multipart.File
		contentType string
		err         error
	)
	src, err = fh.Open()
	if err != nil {
		return usecase.KYCUpload{}, err
	}
	defer src.Close()
	contentType = fh.Header.Get("Content-Type")
	if contentType == "" {
		contentType = detectImageType(fh.Filename)
	}
	return usecase.KYCUpload{
		Reader:      src,
		ContentType: contentType,
		Size:        fh.Size,
	}, nil
}

func detectImageType(name string) string {
	var lower string
	lower = strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".webp"):
		return "image/webp"
	default:
		return ""
	}
}
