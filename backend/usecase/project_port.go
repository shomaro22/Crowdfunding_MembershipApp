package usecase

import (
	"context"

	"crowdfund-app/model"

	"github.com/google/uuid"
)

type IProjectUsecase interface {
	Create(ctx context.Context, sellerID uuid.UUID, in CreateProjectIn) (*ProjectView, error)
	ListPublic(ctx context.Context, search, sort string, limit, offset int) ([]ProjectView, int64, error)
	ListFeatured(ctx context.Context, limit int) ([]ProjectView, error)
	ListMine(ctx context.Context, sellerID uuid.UUID, limit, offset int) ([]ProjectView, int64, error)
	FindView(ctx context.Context, id uuid.UUID, viewerID *uuid.UUID, viewerRole string) (*ProjectView, error)
	ListPlans(ctx context.Context, projectID uuid.UUID, viewerID *uuid.UUID, viewerRole string) ([]ProjectPlanView, error)
	Update(ctx context.Context, sellerID, projectID uuid.UUID, in UpdateProjectIn) (*ProjectView, error)
	RequestDelete(ctx context.Context, sellerID, projectID uuid.UUID) (*ProjectView, error)
	EnrichList(ctx context.Context, rows []model.Project, total int64) ([]ProjectView, int64, error)
}

type ISellerUsecase interface {
	DashboardStats(ctx context.Context, sellerID uuid.UUID) (SellerDashboardStatsView, error)
	ListSupporters(ctx context.Context, sellerID uuid.UUID, projectID *uuid.UUID, limit, offset int) (SellerSupportersResult, error)
	ExportSupporters(ctx context.Context, sellerID uuid.UUID, projectID *uuid.UUID) ([]SupporterView, error)
	ProjectAnalysis(ctx context.Context, sellerID uuid.UUID) (ProjectAnalysisResult, error)
}
