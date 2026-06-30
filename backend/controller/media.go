package controller

import (
	"io"
	"mime/multipart"
	"net/http"

	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type mediaController struct {
	Media usecase.IMediaUsecase
}

func NewMediaController(media usecase.IMediaUsecase) IMediaController {
	return &mediaController{Media: media}
}

func (h *mediaController) UploadImage(c echo.Context) error {
	var (
		userID uuid.UUID
		fh     *multipart.FileHeader
		upload usecase.MediaUploadIn
		url    string
		err    error
	)
	userID, err = requireUserID(c)
	if err != nil {
		return err
	}
	fh, err = c.FormFile("file")
	if err != nil || fh == nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("file required", "BAD_REQUEST"))
	}
	upload, err = readMediaFile(fh)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid file", "BAD_REQUEST"))
	}
	if closer, ok := upload.Reader.(io.Closer); ok {
		defer closer.Close()
	}
	url, err = h.Media.UploadImage(reqCtx(c), userID, upload)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonCreated(c, mediaUploadJSON{URL: url})
}

type mediaUploadJSON struct {
	URL string `json:"url"`
}

func readMediaFile(fh *multipart.FileHeader) (usecase.MediaUploadIn, error) {
	var (
		src         multipart.File
		contentType string
		err         error
	)
	src, err = fh.Open()
	if err != nil {
		return usecase.MediaUploadIn{}, err
	}
	contentType = resolveImageContentType(fh.Header.Get("Content-Type"), fh.Filename)
	return usecase.MediaUploadIn{
		Reader:      src,
		ContentType: contentType,
		Size:        fh.Size,
	}, nil
}
