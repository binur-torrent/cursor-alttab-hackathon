package usecase

import (
	"context"

	"github.com/masterfabric-go/masterfabric/internal/domain/lighting/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/lighting/repository"
)

// GetStatsUseCase computes network-wide lighting KPIs for the dashboard.
type GetStatsUseCase struct {
	segments repository.SegmentRepository
}

// NewGetStatsUseCase creates a new GetStatsUseCase.
func NewGetStatsUseCase(segments repository.SegmentRepository) *GetStatsUseCase {
	return &GetStatsUseCase{segments: segments}
}

// Execute returns aggregated city lighting statistics.
func (uc *GetStatsUseCase) Execute(ctx context.Context) (*model.CityStats, error) {
	return uc.segments.Stats(ctx)
}
