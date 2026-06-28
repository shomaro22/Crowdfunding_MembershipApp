package controller

import (
	"net/http"

	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type FavoriteController struct {
	Favorites FavoriteUseCase
}

func NewFavoriteController(favorites FavoriteUseCase) *FavoriteController {
	return &FavoriteController{Favorites: favorites}
}

func (h *FavoriteController) List(c echo.Context) error {
	var (
		userID uuid.UUID
		rows   []usecase.ProjectView
		err    error
	)
	userID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	rows, err = h.Favorites.List(c.Request().Context(), userID)
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusOK, listFavoritesJSON(rows))
}

func (h *FavoriteController) Toggle(c echo.Context) error {
	var (
		userID uuid.UUID
		body   toggleFavoriteJSON
		out    usecase.FavoriteStatusView
		err    error
	)
	userID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	if err = c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
	projectID, err := uuid.Parse(body.ProjectID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid project_id", "BAD_REQUEST"))
	}
	out, err = h.Favorites.Toggle(c.Request().Context(), userID, usecase.ToggleFavoriteIn{
		ProjectID: projectID,
		Action:    body.Action,
		Meta:      requestMeta(c),
	})
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusOK, favoriteStatusWireFrom(out))
}

func (h *FavoriteController) Check(c echo.Context) error {
	var (
		userID    uuid.UUID
		projectID uuid.UUID
		out       usecase.FavoriteStatusView
		err       error
	)
	userID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	projectID, err = uuid.Parse(c.Param("projectId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid project id", "BAD_REQUEST"))
	}
	out, err = h.Favorites.Check(c.Request().Context(), userID, projectID)
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusOK, favoriteStatusWireFrom(out))
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
