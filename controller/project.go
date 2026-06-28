package controller

import (
	"net/http"
	"strconv"

	"crowdfund-app/middleware"
	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type ProjectController struct {
	Projects ProjectUseCase
}

func NewProjectController(uc ProjectUseCase) *ProjectController {
	return &ProjectController{Projects: uc}
}

func (h *ProjectController) ListPublic(c echo.Context) error {
	var (
		limit  int
		offset int
		rows   []usecase.ProjectView
		total  int64
		err    error
	)
	limit, _ = strconv.Atoi(c.QueryParam("limit"))
	offset, _ = strconv.Atoi(c.QueryParam("offset"))
	rows, total, err = h.Projects.ListPublic(c.Request().Context(), c.QueryParam("search"), c.QueryParam("sort"), limit, offset)
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusOK, listProjectsJSON(rows, total, limit, offset))
}

func (h *ProjectController) Create(c echo.Context) error {
	var (
		sellerID uuid.UUID
		body     createProjectJSON
		in       usecase.CreateProjectIn
		view     *usecase.ProjectView
		err      error
	)
	sellerID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	if err = c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
	in, err = body.toInput()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
	view, err = h.Projects.Create(c.Request().Context(), sellerID, in)
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusCreated, projectDetailJSON(view))
}

func (h *ProjectController) ListMine(c echo.Context) error {
	var (
		sellerID uuid.UUID
		limit    int
		offset   int
		rows     []usecase.ProjectView
		total    int64
		err      error
	)
	sellerID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	limit, _ = strconv.Atoi(c.QueryParam("limit"))
	offset, _ = strconv.Atoi(c.QueryParam("offset"))
	rows, total, err = h.Projects.ListMine(c.Request().Context(), sellerID, limit, offset)
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusOK, listProjectsJSON(rows, total, limit, offset))
}

func (h *ProjectController) Get(c echo.Context) error {
	var (
		id         uuid.UUID
		viewerID   *uuid.UUID
		viewerRole string
		uid        uuid.UUID
		ok         bool
		view       *usecase.ProjectView
		err        error
	)
	id, err = uuid.Parse(c.Param("projectId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid project id", "BAD_REQUEST"))
	}
	if uid, ok = middleware.AuthUserID(c); ok {
		viewerID = &uid
		if role, roleOK := middleware.AuthUserRole(c); roleOK {
			viewerRole = string(role)
		}
	}
	view, err = h.Projects.FindView(c.Request().Context(), id, viewerID, viewerRole)
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusOK, projectDetailJSON(view))
}

func (h *ProjectController) ListPlans(c echo.Context) error {
	var (
		id         uuid.UUID
		viewerID   *uuid.UUID
		viewerRole string
		plans      []usecase.ProjectPlanView
		ok         bool
		uid        uuid.UUID
		err        error
	)
	id, err = uuid.Parse(c.Param("projectId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid project id", "BAD_REQUEST"))
	}
	if uid, ok = middleware.AuthUserID(c); ok {
		viewerID = &uid
		if role, roleOK := middleware.AuthUserRole(c); roleOK {
			viewerRole = string(role)
		}
	}
	plans, err = h.Projects.ListPlans(c.Request().Context(), id, viewerID, viewerRole)
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusOK, projectPlansToJSON(plans))
}

func (h *ProjectController) Update(c echo.Context) error {
	var (
		sellerID uuid.UUID
		id       uuid.UUID
		body     updateProjectJSON
		in       usecase.UpdateProjectIn
		view     *usecase.ProjectView
		err      error
	)
	sellerID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	id, err = uuid.Parse(c.Param("projectId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid project id", "BAD_REQUEST"))
	}
	if err = c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
	in, err = body.toInput()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
	view, err = h.Projects.Update(c.Request().Context(), sellerID, id, in)
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusOK, projectDetailJSON(view))
}

func (h *ProjectController) RequestDelete(c echo.Context) error {
	var (
		sellerID uuid.UUID
		id       uuid.UUID
		view     *usecase.ProjectView
		err      error
	)
	sellerID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	id, err = uuid.Parse(c.Param("projectId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid project id", "BAD_REQUEST"))
	}
	view, err = h.Projects.RequestDelete(c.Request().Context(), sellerID, id)
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusOK, projectDetailJSON(view))
}
