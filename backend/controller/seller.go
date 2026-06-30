package controller

import (
	"encoding/csv"
	"net/http"
	"time"

	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type SellerController struct {
	Seller usecase.ISellerUsecase
}

func NewSellerController(seller usecase.ISellerUsecase) *SellerController {
	return &SellerController{Seller: seller}
}

func (h *SellerController) DashboardStats(c echo.Context) error {
	var (
		sellerID uuid.UUID
		stats    usecase.SellerDashboardStatsView
		err      error
	)
	sellerID, err = requireUserID(c)
	if err != nil {
		return err
	}
	stats, err = h.Seller.DashboardStats(reqCtx(c), sellerID)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, sellerDashboardStatsJSON(stats))
}

func (h *SellerController) ListSupporters(c echo.Context) error {
	var (
		sellerID  uuid.UUID
		projectID *uuid.UUID
		limit     int
		offset    int
		result    usecase.SellerSupportersResult
		err       error
	)
	sellerID, err = requireUserID(c)
	if err != nil {
		return err
	}
	projectID, err = optionalProjectIDQuery(c)
	if err != nil {
		return err
	}
	limit, offset = queryPage(c)
	result, err = h.Seller.ListSupporters(reqCtx(c), sellerID, projectID, limit, offset)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, sellerListSupportersJSON(result))
}

func (h *SellerController) ExportSupporters(c echo.Context) error {
	var (
		sellerID  uuid.UUID
		projectID *uuid.UUID
		rows      []usecase.SupporterView
		err       error
	)
	sellerID, err = requireUserID(c)
	if err != nil {
		return err
	}
	projectID, err = optionalProjectIDQuery(c)
	if err != nil {
		return err
	}
	rows, err = h.Seller.ExportSupporters(reqCtx(c), sellerID, projectID)
	if err != nil {
		return mapUseCaseError(err)
	}
	c.Response().Header().Set("Content-Type", "text/csv; charset=utf-8")
	c.Response().Header().Set("Content-Disposition", `attachment; filename="supporters.csv"`)
	return writeSupportersCSV(c.Response(), rows)
}

func (h *SellerController) ProjectAnalysis(c echo.Context) error {
	var (
		sellerID uuid.UUID
		result   usecase.ProjectAnalysisResult
		err      error
	)
	sellerID, err = requireUserID(c)
	if err != nil {
		return err
	}
	result, err = h.Seller.ProjectAnalysis(reqCtx(c), sellerID)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, projectAnalysisJSON(result))
}

func optionalProjectIDQuery(c echo.Context) (*uuid.UUID, error) {
	var (
		raw string
		id  uuid.UUID
		err error
	)
	raw = c.QueryParam("project_id")
	if raw == "" {
		return nil, nil
	}
	id, err = uuid.Parse(raw)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, apiError("invalid project_id", "BAD_REQUEST"))
	}
	return &id, nil
}

type sellerDashboardStatsWire struct {
	TotalProjects   int64 `json:"total_projects"`
	ActiveProjects  int64 `json:"active_projects"`
	TotalSupporters int64 `json:"total_supporters"`
	TotalRevenue    int64 `json:"total_revenue"`
}

func sellerDashboardStatsJSON(v usecase.SellerDashboardStatsView) sellerDashboardStatsWire {
	return sellerDashboardStatsWire{
		TotalProjects:   v.TotalProjects,
		ActiveProjects:  v.ActiveProjects,
		TotalSupporters: v.TotalSupporters,
		TotalRevenue:    v.TotalRevenue,
	}
}

func sellerListSupportersJSON(result usecase.SellerSupportersResult) listSupportersResponse {
	var (
		items []supporterJSON
		i     int
	)
	items = make([]supporterJSON, 0, len(result.Supporters))
	for i = range result.Supporters {
		items = append(items, supporterToJSON(result.Supporters[i]))
	}
	return listSupportersResponse{Supporters: items, Total: int(result.Total)}
}

type projectAnalysisResponse struct {
	Projects []projectAnalysisItemJSON `json:"projects"`
}

type projectAnalysisItemJSON struct {
	ProjectID      string         `json:"project_id"`
	Title          string         `json:"title"`
	SupporterCount int            `json:"supporter_count"`
	Revenue        int64          `json:"revenue"`
	PlanBreakdown  map[string]int `json:"plan_breakdown"`
}

func projectAnalysisJSON(result usecase.ProjectAnalysisResult) projectAnalysisResponse {
	var (
		out projectAnalysisResponse
		i   int
	)
	out.Projects = make([]projectAnalysisItemJSON, 0, len(result.Projects))
	for i = range result.Projects {
		out.Projects = append(out.Projects, projectAnalysisItemJSON{
			ProjectID:      result.Projects[i].ProjectID.String(),
			Title:          result.Projects[i].Title,
			SupporterCount: result.Projects[i].SupporterCount,
			Revenue:        result.Projects[i].Revenue,
			PlanBreakdown:  result.Projects[i].PlanBreakdown,
		})
	}
	return out
}

func writeSupportersCSV(w http.ResponseWriter, rows []usecase.SupporterView) error {
	var (
		cw   *csv.Writer
		i    int
		v    usecase.SupporterView
		rec  []string
		err  error
	)
	cw = csv.NewWriter(w)
	err = cw.Write([]string{
		"id", "project_id", "project_title", "user_id", "plan_id", "status",
		"billing_mode", "start_date", "end_date",
	})
	if err != nil {
		return err
	}
	for i = range rows {
		v = rows[i]
		rec = []string{
			v.Supporter.ID.String(),
			v.Supporter.ProjectID.String(),
			projectTitle(v.Project),
			v.Supporter.UserID.String(),
			v.Supporter.PlanID.String(),
			v.Supporter.Status,
			v.Supporter.BillingMode,
			v.Supporter.CurrentPeriodStart.UTC().Format(time.RFC3339),
			v.Supporter.CurrentPeriodEnd.UTC().Format(time.RFC3339),
		}
		if err = cw.Write(rec); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func projectTitle(p *usecase.ProjectItemView) string {
	if p == nil {
		return ""
	}
	return p.Title
}
