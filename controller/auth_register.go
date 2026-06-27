package controller

import (
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"crowdfund-app/usecase"

	"github.com/labstack/echo/v4"
)

type registerRequest struct {
	Email    string  `json:"email"`
	Password string  `json:"password"`
	Name     *string `json:"name"`
}

type sellerRegPayload struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	FullName     string `json:"full_name"`
	BirthDate    string `json:"birth_date"`
	Address      string `json:"address"`
	PhotoPresent bool   `json:"photo_present"`
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
	u, err = h.Auth.RegisterBuyer(c.Request().Context(), body.Email, body.Password, body.Name, requestMeta(c))
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusCreated, registerResponse{
		User:    usecase.UserWireFrom(u),
		Message: "Registration pending admin approval",
	})
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
	}, requestMeta(c))
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusCreated, sellerCreateJSON{
		User:         usecase.UserWireFrom(res.User),
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
