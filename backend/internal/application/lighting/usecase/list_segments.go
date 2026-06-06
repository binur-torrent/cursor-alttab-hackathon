// Package usecase contains the lighting context application use cases.
package usecase

import (
	"context"

	"github.com/masterfabric-go/masterfabric/internal/domain/lighting/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/lighting/repository"
)

// ListSegmentsUseCase lists street segments with optional filtering.
type ListSegmentsUseCase struct {
	segments repository.SegmentRepository
}

// NewListSegmentsUseCase creates a new ListSegmentsUseCase.
func NewListSegmentsUseCase(segments repository.SegmentRepository) *ListSegmentsUseCase {
	return &ListSegmentsUseCase{segments: segments}
}

// Execute returns a paginated, filtered list of segments (highest risk first).
func (uc *ListSegmentsUseCase) Execute(ctx context.Context, filter repository.SegmentFilter, offset, limit int) ([]*model.StreetSegment, int, error) {
	return uc.segments.List(ctx, filter, offset, limit)
}

// ExecuteAll returns every matching segment (used by the map view).
func (uc *ListSegmentsUseCase) ExecuteAll(ctx context.Context, filter repository.SegmentFilter) ([]*model.StreetSegment, error) {
	return uc.segments.ListAll(ctx, filter)
}
