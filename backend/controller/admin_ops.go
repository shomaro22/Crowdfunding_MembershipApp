package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type AdminOperationsController struct {
	Operations usecase.IOperationsUsecase
}

func NewAdminOperationsController(ops usecase.IOperationsUsecase) *AdminOperationsController {
	return &AdminOperationsController{Operations: ops}
}

func (h *AdminOperationsController) ListActivity(c echo.Context) error {
	var (
		result usecase.ActivityLogsResult
		q      usecase.ActivityLogsQuery
		err    error
	)
	q, err = activityLogsQueryFrom(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid query", "BAD_REQUEST"))
	}
	result, err = h.Operations.ListActivityLogs(reqCtx(c), q)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, ListActivityLogsWireFrom(result))
}

func (h *AdminOperationsController) ListAuditLogs(c echo.Context) error {
	var (
		result usecase.AuditLogsResult
		q      usecase.AuditLogsQuery
		err    error
	)
	q, err = auditLogsQueryFrom(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid query", "BAD_REQUEST"))
	}
	result, err = h.Operations.ListAuditLogs(reqCtx(c), q)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, ListAuditLogsWireFrom(result))
}

func (h *AdminOperationsController) ListBatches(c echo.Context) error {
	var (
		result usecase.BatchRunsResult
		q      usecase.BatchRunsQuery
		err    error
	)
	q = batchRunsQueryFrom(c)
	result, err = h.Operations.ListBatchRuns(reqCtx(c), q)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, ListBatchJobRunsWireFrom(result))
}

func (h *AdminOperationsController) RunBatch(c echo.Context) error {
	var (
		adminID uuid.UUID
		jobName string
		dryRun  bool
		run     *usecase.BatchRunView
		err     error
	)
	adminID, err = requireUserID(c)
	if err != nil {
		return err
	}
	jobName = strings.TrimSpace(c.Param("jobName"))
	dryRun, err = parseBoolQuery(c.QueryParam("dry_run"), false)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid dry_run", "BAD_REQUEST"))
	}
	run, err = h.Operations.RunBatch(reqCtx(c), adminID, jobName, dryRun)
	if err != nil {
		if run != nil && errors.Is(err, usecase.ErrBatchJobFailed) {
			return c.JSON(http.StatusUnprocessableEntity, BatchJobRunWireFrom(*run))
		}
		return mapUseCaseError(err)
	}
	return jsonOK(c, BatchJobRunWireFrom(*run))
}

func (h *AdminOperationsController) GetSecurity(c echo.Context) error {
	var (
		from   *time.Time
		to     *time.Time
		view   usecase.SecurityEventsView
		err    error
	)
	from, err = parseOptionalRFC3339(c.QueryParam("from"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid from", "BAD_REQUEST"))
	}
	to, err = parseOptionalRFC3339(c.QueryParam("to"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid to", "BAD_REQUEST"))
	}
	view, err = h.Operations.GetSecurityEvents(reqCtx(c), from, to)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, SecurityEventsWireFrom(view))
}

func activityLogsQueryFrom(c echo.Context) (usecase.ActivityLogsQuery, error) {
	var (
		q      usecase.ActivityLogsQuery
		limit  int
		offset int
		err    error
	)
	q.UserID, err = parseOptionalUUID(c.QueryParam("user_id"))
	if err != nil {
		return usecase.ActivityLogsQuery{}, err
	}
	if s := strings.TrimSpace(c.QueryParam("action")); s != "" {
		q.Action = &s
	}
	if s := strings.TrimSpace(c.QueryParam("resource")); s != "" {
		q.Resource = &s
	}
	if s := strings.TrimSpace(c.QueryParam("resource_id")); s != "" {
		q.ResourceID = &s
	}
	if s := strings.TrimSpace(c.QueryParam("ip")); s != "" {
		q.IP = &s
	}
	q.From, err = parseOptionalRFC3339(c.QueryParam("from"))
	if err != nil {
		return usecase.ActivityLogsQuery{}, err
	}
	q.To, err = parseOptionalRFC3339(c.QueryParam("to"))
	if err != nil {
		return usecase.ActivityLogsQuery{}, err
	}
	limit, offset = queryPage(c)
	q.Limit = limit
	q.Offset = offset
	return q, nil
}

func auditLogsQueryFrom(c echo.Context) (usecase.AuditLogsQuery, error) {
	var (
		q      usecase.AuditLogsQuery
		limit  int
		offset int
		err    error
	)
	q.AdminID, err = parseOptionalUUID(c.QueryParam("admin_id"))
	if err != nil {
		return usecase.AuditLogsQuery{}, err
	}
	if s := strings.TrimSpace(c.QueryParam("action")); s != "" {
		q.Action = &s
	}
	if s := strings.TrimSpace(c.QueryParam("resource")); s != "" {
		q.Resource = &s
	}
	limit, offset = queryPage(c)
	q.Limit = limit
	q.Offset = offset
	return q, nil
}

func batchRunsQueryFrom(c echo.Context) usecase.BatchRunsQuery {
	var (
		q     usecase.BatchRunsQuery
		limit int
	)
	if s := strings.TrimSpace(c.QueryParam("job_name")); s != "" {
		q.JobName = &s
	}
	if s := strings.TrimSpace(c.QueryParam("status")); s != "" {
		q.Status = &s
	}
	limit, _ = strconv.Atoi(c.QueryParam("limit"))
	q.Limit = limit
	return q
}

func parseOptionalUUID(raw string) (*uuid.UUID, error) {
	var (
		id  uuid.UUID
		err error
	)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	id, err = uuid.Parse(raw)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func parseOptionalRFC3339(raw string) (*time.Time, error) {
	var (
		t   time.Time
		err error
		utc time.Time
	)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	t, err = time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	utc = t.UTC()
	return &utc, nil
}

func parseBoolQuery(raw string, defaultVal bool) (bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultVal, nil
	}
	return strconv.ParseBool(raw)
}
