package controller

import (
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/storage"
	"crowdfund-app/usecase"

	"github.com/labstack/echo/v4"
)

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

func sellerRegToJSON(r *entity.SellerReg) sellerRegJSON {
	if r == nil {
		return sellerRegJSON{}
	}
	return sellerRegJSON{
		ID:           r.ID.String(),
		UserID:       r.UserID.String(),
		FullName:     r.FullName,
		BirthDate:    r.BirthDate.Format("2006-01-02"),
		Address:      r.Address,
		PhotoPresent: r.PhotoPresent,
		IDDocType:    string(entity.IDDriversLicense),
		HasIDFront:   r.HasIDFront(),
		HasIDBack:    r.HasIDBack(),
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
		frontImg  storage.KYCImage
		backImg   storage.KYCImage
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
		return mapSellerRegError(err)
	}
	backImg, err = readKYCFile(backFile)
	if err != nil {
		return mapSellerRegError(err)
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
		return mapSellerRegError(err)
	}
	return c.JSON(http.StatusCreated, sellerCreateJSON{
		User:         userToJSON(res.User),
		Registration: sellerRegToJSON(res.Registration),
	})
}

func readKYCFile(fh *multipart.FileHeader) (storage.KYCImage, error) {
	var (
		src         multipart.File
		contentType string
		err         error
	)
	src, err = fh.Open()
	if err != nil {
		return storage.KYCImage{}, err
	}
	defer src.Close()
	contentType = fh.Header.Get("Content-Type")
	if contentType == "" {
		contentType = detectImageType(fh.Filename)
	}
	return storage.KYCImage{
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

func mapSellerRegError(err error) *echo.HTTPError {
	switch err {
	case storage.ErrBadMedia:
		return echo.NewHTTPError(http.StatusUnsupportedMediaType, apiError("unsupported image media type", "UNSUPPORTED_MEDIA"))
	case storage.ErrTooLarge:
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, apiError("identity document image too large", "FILE_TOO_LARGE"))
	case entity.ErrBadKYC:
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid seller kyc metadata", "BAD_SELLER_KYC"))
	default:
		return mapAuthError(err)
	}
}
