package usecase

import (
	"context"

	"crowdfund-app/entity"
)

type ConnectSyncer interface {
	SyncFromStripeAccountID(ctx context.Context, stripeAccountID string) (*ConnectAccountView, error)
}

type SupporterViewer interface {
	ViewForSupporter(ctx context.Context, supporter *entity.ProjectSupporter) (SupporterView, error)
}

func (s *SupporterUseCase) ViewForSupporter(ctx context.Context, supporter *entity.ProjectSupporter) (SupporterView, error) {
	return s.enrichView(ctx, supporter)
}
