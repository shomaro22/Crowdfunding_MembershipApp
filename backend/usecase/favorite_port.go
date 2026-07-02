package usecase

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
)

type IFavoriteUsecase interface {
	List(ctx context.Context, userID uuid.UUID) ([]ProjectView, error)
	Toggle(ctx context.Context, userID uuid.UUID, in ToggleFavoriteIn) (FavoriteStatusView, error)
	Check(ctx context.Context, userID, projectID uuid.UUID) (FavoriteStatusView, error)
}

type IReportUsecase interface {
	Create(ctx context.Context, reporterID uuid.UUID, in CreateReportIn) (ReportView, error)
	ListAdmin(ctx context.Context, status *string, limit, offset int) (ReportsResult, error)
	UpdateAdmin(ctx context.Context, adminID, reportID uuid.UUID, in UpdateReportIn) (ReportView, error)
}

type IMediaUsecase interface {
	UploadImage(ctx context.Context, userID uuid.UUID, in MediaUploadIn) (string, error)
}

type IKYCAccessUsecase interface {
	OpenWithToken(ctx context.Context, token string) (io.ReadCloser, string, error)
	SignToken(storageKey string, now time.Time) string
}

type IRateLimitUsecase interface {
	Allow(ctx context.Context, key string, limit int, period time.Duration) (bool, int, error)
}
