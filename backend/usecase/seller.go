package usecase

import (
	"context"

	"crowdfund-app/model"
	"crowdfund-app/repository"
	"crowdfund-app/validator"

	"github.com/google/uuid"
)

type SellerDashboardStatsView struct {
	TotalProjects   int64
	ActiveProjects  int64
	TotalSupporters int64
	TotalRevenue    int64
}

type SellerSupportersResult struct {
	Supporters []SupporterView
	Total      int64
}

type ProjectAnalysisItemView struct {
	ProjectID      uuid.UUID
	Title          string
	SupporterCount int
	Revenue        int64
	PlanBreakdown  map[string]int
}

type ProjectAnalysisResult struct {
	Projects []ProjectAnalysisItemView
}

type SupporterEnricher interface {
	EnrichSupporterRows(ctx context.Context, rows []model.ProjectSupporter) ([]SupporterView, error)
}

type SellerUseCase struct {
	Projects   repository.IProjectRepo
	Seller     repository.ISellerRepo
	Supporters SupporterEnricher
}

func (s *SellerUseCase) DashboardStats(ctx context.Context, sellerID uuid.UUID) (SellerDashboardStatsView, error) {
	var (
		stats repository.SellerDashboardStats
		err   error
	)
	if s == nil || s.Seller == nil {
		return SellerDashboardStatsView{}, ErrBadRequest
	}
	stats, err = s.Seller.DashboardStats(ctx, sellerID)
	if err != nil {
		return SellerDashboardStatsView{}, err
	}
	return SellerDashboardStatsView{
		TotalProjects:   stats.TotalProjects,
		ActiveProjects:  stats.ActiveProjects,
		TotalSupporters: stats.TotalSupporters,
		TotalRevenue:    stats.TotalRevenue,
	}, nil
}

func (s *SellerUseCase) ListSupporters(ctx context.Context, sellerID uuid.UUID, projectID *uuid.UUID, limit, offset int) (SellerSupportersResult, error) {
	var (
		norm  validator.SellerSupporterListNormalized
		rows  []model.ProjectSupporter
		views []SupporterView
		total int64
		err   error
	)
	if s == nil || s.Seller == nil || s.Supporters == nil {
		return SellerSupportersResult{}, ErrBadRequest
	}
	norm, err = validator.ValidateSellerSupporterList(validator.SellerSupporterListInput{
		ProjectID: projectID,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return SellerSupportersResult{}, ErrBadRequest
	}
	if norm.ProjectID != nil {
		if err = s.ensureSellerProject(ctx, sellerID, *norm.ProjectID); err != nil {
			return SellerSupportersResult{}, err
		}
	}
	rows, total, err = s.Seller.ListSupporters(ctx, sellerID, norm.ProjectID, norm.Limit, norm.Offset)
	if err != nil {
		return SellerSupportersResult{}, err
	}
	views, err = s.Supporters.EnrichSupporterRows(ctx, rows)
	if err != nil {
		return SellerSupportersResult{}, err
	}
	return SellerSupportersResult{Supporters: views, Total: total}, nil
}

func (s *SellerUseCase) ExportSupporters(ctx context.Context, sellerID uuid.UUID, projectID *uuid.UUID) ([]SupporterView, error) {
	var (
		rows  []model.ProjectSupporter
		total int64
		err   error
	)
	if s == nil || s.Seller == nil || s.Supporters == nil {
		return nil, ErrBadRequest
	}
	if projectID != nil {
		if err = s.ensureSellerProject(ctx, sellerID, *projectID); err != nil {
			return nil, err
		}
	}
	rows, total, err = s.Seller.ListSupporters(ctx, sellerID, projectID, repository.MaxSellerSupporterExport, 0)
	if err != nil {
		return nil, err
	}
	_ = total
	return s.Supporters.EnrichSupporterRows(ctx, rows)
}

func (s *SellerUseCase) ProjectAnalysis(ctx context.Context, sellerID uuid.UUID) (ProjectAnalysisResult, error) {
	var (
		rows       []repository.ProjectAnalysisRow
		breakdowns []repository.PlanBreakdownRow
		byProject  map[uuid.UUID]map[string]int
		out        ProjectAnalysisResult
		i          int
		row        repository.ProjectAnalysisRow
		bd         repository.PlanBreakdownRow
		err        error
	)
	if s == nil || s.Seller == nil {
		return ProjectAnalysisResult{}, ErrBadRequest
	}
	rows, err = s.Seller.ListProjectAnalysis(ctx, sellerID)
	if err != nil {
		return ProjectAnalysisResult{}, err
	}
	breakdowns, err = s.Seller.PlanBreakdownByProject(ctx, sellerID)
	if err != nil {
		return ProjectAnalysisResult{}, err
	}
	byProject = map[uuid.UUID]map[string]int{}
	for _, bd = range breakdowns {
		if byProject[bd.ProjectID] == nil {
			byProject[bd.ProjectID] = map[string]int{}
		}
		byProject[bd.ProjectID][bd.MembershipType] = bd.Count
	}
	out.Projects = make([]ProjectAnalysisItemView, 0, len(rows))
	for i = range rows {
		row = rows[i]
		out.Projects = append(out.Projects, ProjectAnalysisItemView{
			ProjectID:      row.ProjectID,
			Title:          row.Title,
			SupporterCount: row.SupporterCount,
			Revenue:        row.Revenue,
			PlanBreakdown:  clonePlanBreakdown(byProject[row.ProjectID]),
		})
	}
	return out, nil
}

func (s *SellerUseCase) ensureSellerProject(ctx context.Context, sellerID, projectID uuid.UUID) error {
	var (
		project model.Project
		err     error
	)
	if s.Projects == nil {
		return ErrBadRequest
	}
	project, _, _, err = s.Projects.GetID(ctx, projectID)
	if err != nil {
		return mapRepoNotFoundAs(err, ErrForbidden)
	}
	if project.SellerID != sellerID {
		return ErrForbidden
	}
	return nil
}

func clonePlanBreakdown(src map[string]int) map[string]int {
	var (
		out map[string]int
		k   string
		v   int
	)
	if len(src) == 0 {
		return map[string]int{}
	}
	out = make(map[string]int, len(src))
	for k, v = range src {
		out[k] = v
	}
	return out
}
