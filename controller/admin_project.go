package controller

import (
	"net/http"
	"strconv"

	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (h *AdminController) ListProjects(c echo.Context) error {
	var (
		status *string
		s      string
		limit  int
		offset int
		rows   []usecase.ProjectView
		total  int64
		err    error
	)
	s = c.QueryParam("status")
	if s != "" {
		status = &s
	}
	limit, _ = strconv.Atoi(c.QueryParam("limit"))
	offset, _ = strconv.Atoi(c.QueryParam("offset"))
	rows, total, err = h.Admin.ListProjects(c.Request().Context(), status, limit, offset)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, listProjectsJSON(rows, total, limit, offset))
}

func (h *AdminController) PatchProject(c echo.Context) error {
	var (
		adminID uuid.UUID
		id      uuid.UUID
		body    struct {
			Status string  `json:"status"`
			Reason *string `json:"reason"`
		}
		view *usecase.ProjectView
		err  error
	)
	adminID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	id, err = uuid.Parse(c.Param("projectId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid project id", "BAD_REQUEST"))
	}
	if err = c.Bind(&body); err != nil || body.Status == "" {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("status required", "BAD_REQUEST"))
	}
	view, err = h.Admin.PatchProject(c.Request().Context(), adminID, id, body.Status, body.Reason)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, projectDetailJSON(view))
}

func (h *AdminController) DeleteProject(c echo.Context) error {
	var (
		adminID uuid.UUID
		id      uuid.UUID
		err     error
	)
	adminID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	id, err = uuid.Parse(c.Param("projectId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid project id", "BAD_REQUEST"))
	}
	if err = h.Admin.DeleteProject(c.Request().Context(), adminID, id); err != nil {
		return mapAdminError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *AdminController) PatchDeleteReq(c echo.Context) error {
	var (
		adminID uuid.UUID
		id      uuid.UUID
		body    struct {
			Action string  `json:"action"`
			Reason *string `json:"reason"`
		}
		view *usecase.ProjectView
		err  error
	)
	adminID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	id, err = uuid.Parse(c.Param("projectId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid project id", "BAD_REQUEST"))
	}
	if err = c.Bind(&body); err != nil || body.Action == "" {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("action required", "BAD_REQUEST"))
	}
	view, err = h.Admin.PatchDeleteReq(c.Request().Context(), adminID, id, body.Action, body.Reason)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, projectDetailJSON(view))
}
