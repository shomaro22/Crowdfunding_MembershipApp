package controller

import (
	"net/http"

	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type projectController struct {
	Projects usecase.IProjectUsecase
}

func NewProjectController(uc usecase.IProjectUsecase) IProjectController {
	return &projectController{Projects: uc}
}

func (h *projectController) ListPublic(c echo.Context) error {
	var (
		limit  int
		offset int
		rows   []usecase.ProjectView
		total  int64
		err    error
	)
	limit, offset = queryPage(c)
	rows, total, err = h.Projects.ListPublic(reqCtx(c), c.QueryParam("search"), c.QueryParam("sort"), limit, offset)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, listProjectsJSON(rows, total, limit, offset))
}

func (h *projectController) ListFeatured(c echo.Context) error {
	var (
		limit int
		rows  []usecase.ProjectView
		err   error
	)
	limit = queryLimit(c, 5, 20)
	rows, err = h.Projects.ListFeatured(reqCtx(c), limit)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, listProjectsJSON(rows, int64(len(rows)), limit, 0))
}

func (h *projectController) Create(c echo.Context) error {
	var (
		sellerID uuid.UUID
		body     createProjectJSON
		in       usecase.CreateProjectIn
		view     *usecase.ProjectView
		err      error
	)
	sellerID, err = requireUserID(c)
	if err != nil {
		return err
	}
	if err = bindBody(c, &body); err != nil {
		return err
	}
	in, err = body.toInput()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
	view, err = h.Projects.Create(reqCtx(c), sellerID, in)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonCreated(c, projectDetailJSON(view))
}

func (h *projectController) ListMine(c echo.Context) error {
	var (
		sellerID uuid.UUID
		limit    int
		offset   int
		rows     []usecase.ProjectView
		total    int64
		err      error
	)
	sellerID, err = requireUserID(c)
	if err != nil {
		return err
	}
	limit, offset = queryPage(c)
	rows, total, err = h.Projects.ListMine(reqCtx(c), sellerID, limit, offset)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, listProjectsJSON(rows, total, limit, offset))
}

func (h *projectController) Get(c echo.Context) error {
	var (
		id   uuid.UUID
		view *usecase.ProjectView
		v    viewerCtx
		err  error
	)
	id, err = pathUUID(c, "projectId", "invalid project id")
	if err != nil {
		return err
	}
	v = optionalViewer(c)
	view, err = h.Projects.FindView(reqCtx(c), id, v.UserID, v.Role)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, projectDetailJSON(view))
}

func (h *projectController) ListPlans(c echo.Context) error {
	var (
		id    uuid.UUID
		plans []usecase.ProjectPlanView
		v     viewerCtx
		err   error
	)
	id, err = pathUUID(c, "projectId", "invalid project id")
	if err != nil {
		return err
	}
	v = optionalViewer(c)
	plans, err = h.Projects.ListPlans(reqCtx(c), id, v.UserID, v.Role)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, projectPlansToJSON(plans))
}

func (h *projectController) Update(c echo.Context) error {
	var (
		sellerID uuid.UUID
		id       uuid.UUID
		body     updateProjectJSON
		in       usecase.UpdateProjectIn
		view     *usecase.ProjectView
		err      error
	)
	sellerID, err = requireUserID(c)
	if err != nil {
		return err
	}
	id, err = pathUUID(c, "projectId", "invalid project id")
	if err != nil {
		return err
	}
	if err = bindBody(c, &body); err != nil {
		return err
	}
	in, err = body.toInput()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
	view, err = h.Projects.Update(reqCtx(c), sellerID, id, in)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, projectDetailJSON(view))
}

func (h *projectController) RequestDelete(c echo.Context) error {
	var (
		sellerID uuid.UUID
		id       uuid.UUID
		view     *usecase.ProjectView
		err      error
	)
	sellerID, err = requireUserID(c)
	if err != nil {
		return err
	}
	id, err = pathUUID(c, "projectId", "invalid project id")
	if err != nil {
		return err
	}
	view, err = h.Projects.RequestDelete(reqCtx(c), sellerID, id)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, projectDetailJSON(view))
}
