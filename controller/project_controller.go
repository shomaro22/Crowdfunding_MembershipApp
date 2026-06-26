package controller

import (
	"context"
	"net/http"
	"strconv"

	"crowdfund-app/entity"
	"crowdfund-app/middleware"
	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type ProjectUseCase interface {
	Create(ctx context.Context, sellerID uuid.UUID, in usecase.CreateProjectIn) (*usecase.ProjectView, error)
	ListPublic(ctx context.Context, search, sort string, limit, offset int) ([]usecase.ProjectView, int64, error)
	ListMine(ctx context.Context, sellerID uuid.UUID, limit, offset int) ([]usecase.ProjectView, int64, error)
	FindView(ctx context.Context, id uuid.UUID, viewerID *uuid.UUID, viewerRole entity.UserRole) (*usecase.ProjectView, error)
	ListPlans(ctx context.Context, projectID uuid.UUID, viewerID *uuid.UUID, viewerRole entity.UserRole) ([]entity.ProjectPlan, error)
	Update(ctx context.Context, sellerID, projectID uuid.UUID, in usecase.UpdateProjectIn) (*usecase.ProjectView, error)
	RequestDelete(ctx context.Context, sellerID, projectID uuid.UUID) (*usecase.ProjectView, error)
}

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
		return mapProjectError(err)
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
		return mapProjectError(err)
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
		return mapProjectError(err)
	}
	return c.JSON(http.StatusOK, listProjectsJSON(rows, total, limit, offset))
}

func (h *ProjectController) Get(c echo.Context) error {
	var (
		id       uuid.UUID
		viewerID *uuid.UUID
		role     entity.UserRole
		uid      uuid.UUID
		ok       bool
		r        entity.UserRole
		view     *usecase.ProjectView
		err      error
	)
	id, err = uuid.Parse(c.Param("projectId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid project id", "BAD_REQUEST"))
	}
	uid, ok = middleware.AuthUserID(c)
	if ok {
		viewerID = &uid
		r, ok = middleware.AuthUserRole(c)
		if ok {
			role = r
		}
	}
	view, err = h.Projects.FindView(c.Request().Context(), id, viewerID, role)
	if err != nil {
		return mapProjectError(err)
	}
	return c.JSON(http.StatusOK, projectDetailJSON(view))
}

func (h *ProjectController) ListPlans(c echo.Context) error {
	var (
		id       uuid.UUID
		viewerID *uuid.UUID
		role     entity.UserRole
		plans    []entity.ProjectPlan
		ok       bool
		uid      uuid.UUID
		r        entity.UserRole
		err      error
	)
	id, err = uuid.Parse(c.Param("projectId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid project id", "BAD_REQUEST"))
	}
	if uid, ok = middleware.AuthUserID(c); ok {
		viewerID = &uid
		if r, ok = middleware.AuthUserRole(c); ok {
			role = r
		}
	}
	plans, err = h.Projects.ListPlans(c.Request().Context(), id, viewerID, role)
	if err != nil {
		return mapProjectError(err)
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
	in = body.toInput()
	view, err = h.Projects.Update(c.Request().Context(), sellerID, id, in)
	if err != nil {
		return mapProjectError(err)
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
		return mapProjectError(err)
	}
	return c.JSON(http.StatusOK, projectDetailJSON(view))
}

func mapProjectError(err error) *echo.HTTPError {
	switch err {
	case usecase.ErrForbidden:
		return echo.NewHTTPError(http.StatusForbidden, apiError("forbidden", "FORBIDDEN"))
	case usecase.ErrInvalidState:
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid state", "INVALID_STATE"))
	case usecase.ErrBadRequest:
		return echo.NewHTTPError(http.StatusBadRequest, apiError("bad request", "BAD_REQUEST"))
	default:
		return mapAuthError(err)
	}
}
