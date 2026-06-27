package controller

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"crowdfund-app/middleware"
	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type ProjectUseCase interface {
	Create(ctx context.Context, sellerID uuid.UUID, in usecase.CreateProjectIn) (*usecase.ProjectView, error)
	ListPublic(ctx context.Context, search, sort string, limit, offset int) ([]usecase.ProjectView, int64, error)
	ListMine(ctx context.Context, sellerID uuid.UUID, limit, offset int) ([]usecase.ProjectView, int64, error)
	FindView(ctx context.Context, id uuid.UUID, viewerID *uuid.UUID, viewerRole string) (*usecase.ProjectView, error)
	ListPlans(ctx context.Context, projectID uuid.UUID, viewerID *uuid.UUID, viewerRole string) ([]usecase.ProjectPlanView, error)
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
		return mapProjectError(err)
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
	in, err = body.toInput()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
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
	return mapUseCaseError(err)
}

type createProjectJSON struct {
	Title        string               `json:"title"`
	Description  string               `json:"description"`
	TargetAmount int64                `json:"target_amount"`
	DurationType string               `json:"duration_type"`
	StartDate    string               `json:"start_date"`
	EndDate      *string              `json:"end_date"`
	ImageURL     *string              `json:"image_url"`
	Images       []projectImageInJSON `json:"images"`
	Plans        []planInJSON         `json:"plans"`
}

type updateProjectJSON struct {
	Title        *string               `json:"title"`
	Description  *string               `json:"description"`
	TargetAmount *int64                `json:"target_amount"`
	DurationType *string               `json:"duration_type"`
	StartDate    *string               `json:"start_date"`
	EndDate      *string               `json:"end_date"`
	ImageURL     *string               `json:"image_url"`
	Images       *[]projectImageInJSON `json:"images"`
	IsReapply    bool                  `json:"is_reapply"`
}

type projectImageInJSON struct {
	ImageURL string `json:"image_url"`
	IsMain   bool   `json:"is_main"`
	Order    int    `json:"order"`
}

type planInJSON struct {
	MembershipType string `json:"membership_type"`
	Price          int    `json:"price"`
	MaxSupporters  *int   `json:"max_supporters"`
	Benefits       string `json:"benefits"`
}

func (b createProjectJSON) toInput() (usecase.CreateProjectIn, error) {
	var (
		start time.Time
		err   error
		end   *time.Time
		t     time.Time
		plans []usecase.PlanIn
		i     int
		p     planInJSON
	)
	start, err = time.Parse(time.RFC3339, b.StartDate)
	if err != nil {
		return usecase.CreateProjectIn{}, fmt.Errorf("invalid start_date: %w", err)
	}
	if b.EndDate != nil && *b.EndDate != "" {
		t, err = time.Parse(time.RFC3339, *b.EndDate)
		if err != nil {
			return usecase.CreateProjectIn{}, fmt.Errorf("invalid end_date: %w", err)
		}
		end = &t
	}
	plans = make([]usecase.PlanIn, 0, len(b.Plans))
	for i = range b.Plans {
		p = b.Plans[i]
		plans = append(plans, usecase.PlanIn{
			MembershipType: p.MembershipType,
			Price:          p.Price,
			MaxSupporters:  p.MaxSupporters,
			Benefits:       p.Benefits,
		})
	}
	return usecase.CreateProjectIn{
		Title:        b.Title,
		Description:  b.Description,
		TargetAmount: b.TargetAmount,
		DurationType: b.DurationType,
		StartDate:    start,
		EndDate:      end,
		ImageURL:     b.ImageURL,
		Images:       projectImagesInToUsecase(b.Images),
		Plans:        plans,
	}, nil
}

func (b updateProjectJSON) toInput() (usecase.UpdateProjectIn, error) {
	var (
		in     usecase.UpdateProjectIn
		tStart time.Time
		tEnd   time.Time
		err    error
	)
	in = usecase.UpdateProjectIn{
		Title:        b.Title,
		Description:  b.Description,
		TargetAmount: b.TargetAmount,
		ImageURL:     b.ImageURL,
		IsReapply:    b.IsReapply,
	}
	if b.Images != nil {
		imgs := projectImagesInToUsecase(*b.Images)
		in.Images = &imgs
	}
	if b.DurationType != nil {
		in.DurationType = b.DurationType
	}
	if b.StartDate != nil {
		tStart, err = time.Parse(time.RFC3339, *b.StartDate)
		if err != nil {
			return usecase.UpdateProjectIn{}, fmt.Errorf("invalid start_date: %w", err)
		}
		in.StartDate = &tStart
	}
	if b.EndDate != nil {
		if *b.EndDate == "" {
			in.EndDate = nil
		} else {
			tEnd, err = time.Parse(time.RFC3339, *b.EndDate)
			if err != nil {
				return usecase.UpdateProjectIn{}, fmt.Errorf("invalid end_date: %w", err)
			}
			in.EndDate = &tEnd
		}
	}
	return in, nil
}

func projectImagesInToUsecase(rows []projectImageInJSON) []usecase.ImageIn {
	var (
		out []usecase.ImageIn
		i   int
	)
	out = make([]usecase.ImageIn, 0, len(rows))
	for i = range rows {
		out = append(out, usecase.ImageIn{
			ImageURL: rows[i].ImageURL,
			IsMain:   rows[i].IsMain,
			Order:    rows[i].Order,
		})
	}
	return out
}
