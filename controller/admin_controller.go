package controller

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/storage"
	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type AdminUseCase interface {
	ListBuyerPending(ctx context.Context, limit, offset int) ([]entity.User, int64, error)
	ApproveBuyer(ctx context.Context, adminID, userID uuid.UUID, reason *string) (*entity.User, error)
	RejectBuyer(ctx context.Context, adminID, userID uuid.UUID, reason *string) (*entity.User, error)
	ListSellerRegs(ctx context.Context, status entity.RegStatus, limit, offset int) ([]entity.SellerReg, int64, error)
	GetSellerReg(ctx context.Context, id uuid.UUID) (*usecase.SellerRegView, error)
	ApproveSellerReg(ctx context.Context, adminID, regID uuid.UUID) (*entity.SellerReg, error)
	RejectSellerReg(ctx context.Context, adminID, regID uuid.UUID, reason string) (*entity.SellerReg, error)
}

type AdminController struct {
	Admin AdminUseCase
}

func NewAdminController(admin AdminUseCase) *AdminController {
	return &AdminController{Admin: admin}
}

func (h *AdminController) ListBuyerPending(c echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	users, total, err := h.Admin.ListBuyerPending(c.Request().Context(), limit, offset)
	if err != nil {
		return mapAdminError(err)
	}
	items := make([]userJSON, 0, len(users))
	for i := range users {
		items = append(items, userToJSON(&users[i]))
	}
	return c.JSON(http.StatusOK, map[string]any{"users": items, "total": total})
}

func (h *AdminController) ApproveBuyer(c echo.Context) error {
	adminID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid user id", "BAD_REQUEST"))
	}
	u, err := h.Admin.ApproveBuyer(c.Request().Context(), adminID, userID, nil)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, userToJSON(u))
}

func (h *AdminController) RejectBuyer(c echo.Context) error {
	adminID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid user id", "BAD_REQUEST"))
	}
	var body struct {
		Reason *string `json:"reason"`
	}
	_ = c.Bind(&body)
	u, err := h.Admin.RejectBuyer(c.Request().Context(), adminID, userID, body.Reason)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, userToJSON(u))
}

func (h *AdminController) ListSellerRegs(c echo.Context) error {
	status := entity.RegStatus(c.QueryParam("status"))
	if status == "" {
		status = entity.RegPending
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	regs, total, err := h.Admin.ListSellerRegs(c.Request().Context(), status, limit, offset)
	if err != nil {
		return mapAdminError(err)
	}
	items := make([]sellerRegJSON, 0, len(regs))
	for i := range regs {
		items = append(items, sellerRegToJSON(&regs[i]))
	}
	return c.JSON(http.StatusOK, map[string]any{"registrations": items, "total": total})
}

func (h *AdminController) GetSellerReg(c echo.Context) error {
	id, err := uuid.Parse(c.Param("registrationId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid registration id", "BAD_REQUEST"))
	}
	detail, err := h.Admin.GetSellerReg(c.Request().Context(), id)
	if err != nil {
		return mapAdminError(err)
	}
	out := sellerRegToJSON(detail.Registration)
	resp := map[string]any{
		"id":                          out.ID,
		"user_id":                     out.UserID,
		"full_name":                   out.FullName,
		"birth_date":                  out.BirthDate,
		"address":                     out.Address,
		"photo_present":               out.PhotoPresent,
		"identity_document_type":      out.IDDocType,
		"has_identity_document_front": out.HasIDFront,
		"has_identity_document_back":  out.HasIDBack,
		"status":                      out.Status,
		"rejection_reason":            out.RejectReason,
		"created_at":                  out.CreatedAt,
		"updated_at":                  out.UpdatedAt,
		"user":                        userToJSON(detail.User),
	}
	if detail.FrontURL != nil {
		resp["identity_document_front_url"] = *detail.FrontURL
	}
	if detail.BackURL != nil {
		resp["identity_document_back_url"] = *detail.BackURL
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *AdminController) ApproveSellerReg(c echo.Context) error {
	adminID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Param("registrationId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid registration id", "BAD_REQUEST"))
	}
	reg, err := h.Admin.ApproveSellerReg(c.Request().Context(), adminID, id)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, sellerRegToJSON(reg))
}

func (h *AdminController) RejectSellerReg(c echo.Context) error {
	adminID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Param("registrationId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid registration id", "BAD_REQUEST"))
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err := c.Bind(&body); err != nil || body.Reason == "" {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("reason required", "BAD_REQUEST"))
	}
	reg, err := h.Admin.RejectSellerReg(c.Request().Context(), adminID, id, body.Reason)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, sellerRegToJSON(reg))
}

type KYCFileController struct {
	KYC        storage.KYCStore
	SignSecret []byte
}

func NewKYCFileController(kyc storage.KYCStore, secret []byte) *KYCFileController {
	return &KYCFileController{KYC: kyc, SignSecret: secret}
}

func (h *KYCFileController) Serve(c echo.Context) error {
	token := c.QueryParam("token")
	key, err := storage.VerifyKYCAccess(h.SignSecret, token, time.Now().UTC())
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, apiError("invalid or expired token", "INVALID_TOKEN"))
	}
	f, ct, err := h.KYC.Open(key)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, apiError("not found", "NOT_FOUND"))
	}
	defer f.Close()
	return c.Stream(http.StatusOK, ct, f)
}

func mapAdminError(err error) *echo.HTTPError {
	switch {
	case errors.Is(err, usecase.ErrInvalidState):
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid state", "INVALID_STATE"))
	case errors.Is(err, usecase.ErrBadRequest):
		return echo.NewHTTPError(http.StatusBadRequest, apiError("bad request", "BAD_REQUEST"))
	default:
		return mapAuthError(err)
	}
}
