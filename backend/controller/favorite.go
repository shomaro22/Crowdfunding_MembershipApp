package controller

import (
	"net/http"

	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type favoriteController struct {
	Favorites usecase.IFavoriteUsecase
}

func NewFavoriteController(favorites usecase.IFavoriteUsecase) IFavoriteController {
	return &favoriteController{Favorites: favorites}
}

func (h *favoriteController) List(c echo.Context) error {
	var (
		userID uuid.UUID
		rows   []usecase.ProjectView
		err    error
	)
	userID, err = requireUserID(c)
	if err != nil {
		return err
	}
	rows, err = h.Favorites.List(reqCtx(c), userID)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, listFavoritesJSON(rows))
}

func (h *favoriteController) Toggle(c echo.Context) error {
	var (
		userID uuid.UUID
		body   toggleFavoriteJSON
		out    usecase.FavoriteStatusView
		err    error
	)
	userID, err = requireUserID(c)
	if err != nil {
		return err
	}
	if err = bindBody(c, &body); err != nil {
		return err
	}
	projectID, err := uuid.Parse(body.ProjectID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid project_id", "BAD_REQUEST"))
	}
	out, err = h.Favorites.Toggle(reqCtx(c), userID, usecase.ToggleFavoriteIn{
		ProjectID: projectID,
		Action:    body.Action,
		Meta:      requestMeta(c),
	})
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, favoriteStatusWireFrom(out))
}

func (h *favoriteController) Check(c echo.Context) error {
	var (
		userID    uuid.UUID
		projectID uuid.UUID
		out       usecase.FavoriteStatusView
		err       error
	)
	userID, err = requireUserID(c)
	if err != nil {
		return err
	}
	projectID, err = pathUUID(c, "projectId", "invalid project id")
	if err != nil {
		return err
	}
	out, err = h.Favorites.Check(reqCtx(c), userID, projectID)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, favoriteStatusWireFrom(out))
}

type toggleFavoriteJSON struct {
	ProjectID string `json:"project_id"`
	Action    string `json:"action"`
}

type favoriteStatusJSON struct {
	ProjectID   string `json:"project_id"`
	IsFavorited bool   `json:"is_favorited"`
}

type listFavoritesResponse struct {
	Favorites []projectSummaryJSON `json:"favorites"`
}

func listFavoritesJSON(rows []usecase.ProjectView) listFavoritesResponse {
	var (
		out listFavoritesResponse
		i   int
	)
	out.Favorites = make([]projectSummaryJSON, 0, len(rows))
	for i = range rows {
		out.Favorites = append(out.Favorites, projectSummaryToJSON(&rows[i]))
	}
	return out
}

func favoriteStatusWireFrom(v usecase.FavoriteStatusView) favoriteStatusJSON {
	return favoriteStatusJSON{
		ProjectID:   v.ProjectID.String(),
		IsFavorited: v.IsFavorited,
	}
}
