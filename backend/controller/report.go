package controller

import (
	"net/http"
	"time"

	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type reportController struct {
	Reports usecase.IReportUsecase
}

func NewReportController(reports usecase.IReportUsecase) IReportController {
	return &reportController{Reports: reports}
}

func (h *reportController) Create(c echo.Context) error {
	var (
		userID uuid.UUID
		body   createReportJSON
		view   usecase.ReportView
		err    error
	)
	userID, err = requireUserID(c)
	if err != nil {
		return err
	}
	if err = bindBody(c, &body); err != nil {
		return err
	}
	targetID, err := uuid.Parse(body.TargetID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid target_id", "BAD_REQUEST"))
	}
	view, err = h.Reports.Create(reqCtx(c), userID, usecase.CreateReportIn{
		TargetType: body.TargetType,
		TargetID:   targetID,
		Reason:     body.Reason,
		Detail:     body.Detail,
		Meta:       requestMeta(c),
	})
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonCreated(c, reportWireFrom(view))
}

func (h *reportController) ListAdmin(c echo.Context) error {
	var (
		status *string
		s      string
		limit  int
		offset int
		result usecase.ReportsResult
		err    error
	)
	s = c.QueryParam("status")
	if s != "" {
		status = &s
	}
	limit, offset = queryPage(c)
	result, err = h.Reports.ListAdmin(reqCtx(c), status, limit, offset)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, listReportsWireFrom(result))
}

func (h *reportController) PatchAdmin(c echo.Context) error {
	var (
		adminID  uuid.UUID
		reportID uuid.UUID
		body     patchReportJSON
		view     usecase.ReportView
		err      error
	)
	adminID, err = requireUserID(c)
	if err != nil {
		return err
	}
	reportID, err = pathUUID(c, "reportId", "invalid report id")
	if err != nil {
		return err
	}
	if err = bindBody(c, &body); err != nil || body.Status == "" {
		return badRequest(c, "status required")
	}
	view, err = h.Reports.UpdateAdmin(reqCtx(c), adminID, reportID, usecase.UpdateReportIn{
		Status:     body.Status,
		Resolution: body.Resolution,
		Reason:     body.Reason,
	})
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, reportWireFrom(view))
}

type createReportJSON struct {
	TargetType string  `json:"target_type"`
	TargetID   string  `json:"target_id"`
	Reason     string  `json:"reason"`
	Detail     *string `json:"detail"`
}

type patchReportJSON struct {
	Status     string  `json:"status"`
	Resolution *string `json:"resolution"`
	Reason     *string `json:"reason"`
}

type reportWire struct {
	ID         string  `json:"id"`
	ReporterID string  `json:"reporter_id"`
	TargetType string  `json:"target_type"`
	TargetID   string  `json:"target_id"`
	Reason     string  `json:"reason"`
	Detail     *string `json:"detail,omitempty"`
	Status     string  `json:"status"`
	ReviewedBy *string `json:"reviewed_by,omitempty"`
	ReviewedAt *string `json:"reviewed_at,omitempty"`
	Resolution *string `json:"resolution,omitempty"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

type listReportsWire struct {
	Reports []reportWire `json:"reports"`
	Total   int64        `json:"total"`
	Limit   int          `json:"limit"`
	Offset  int          `json:"offset"`
}

func reportWireFrom(v usecase.ReportView) reportWire {
	var (
		out        reportWire
		reviewedBy *string
		reviewedAt *string
		s          string
	)
	out = reportWire{
		ID:         v.ID.String(),
		ReporterID: v.ReporterID.String(),
		TargetType: v.TargetType,
		TargetID:   v.TargetID.String(),
		Reason:     v.Reason,
		Detail:     v.Detail,
		Status:     v.Status,
		Resolution: v.Resolution,
		CreatedAt:  v.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:  v.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if v.ReviewedBy != nil {
		s = v.ReviewedBy.String()
		reviewedBy = &s
		out.ReviewedBy = reviewedBy
	}
	if v.ReviewedAt != nil {
		s = v.ReviewedAt.UTC().Format(time.RFC3339)
		reviewedAt = &s
		out.ReviewedAt = reviewedAt
	}
	return out
}

func listReportsWireFrom(r usecase.ReportsResult) listReportsWire {
	var (
		out listReportsWire
		i   int
	)
	out = listReportsWire{
		Reports: make([]reportWire, len(r.Reports)),
		Total:   r.Total,
		Limit:   r.Limit,
		Offset:  r.Offset,
	}
	for i = range r.Reports {
		out.Reports[i] = reportWireFrom(r.Reports[i])
	}
	return out
}
