package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/application/lighting/dto"
	"github.com/masterfabric-go/masterfabric/internal/domain/lighting/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/lighting/repository"
)

// GetSegmentUseCase fetches a segment with its fixtures and anonymized frames.
type GetSegmentUseCase struct {
	segments repository.SegmentRepository
	fixtures repository.FixtureRepository
	analyses repository.AnalysisRepository
}

// NewGetSegmentUseCase creates a new GetSegmentUseCase.
func NewGetSegmentUseCase(
	segments repository.SegmentRepository,
	fixtures repository.FixtureRepository,
	analyses repository.AnalysisRepository,
) *GetSegmentUseCase {
	return &GetSegmentUseCase{segments: segments, fixtures: fixtures, analyses: analyses}
}

// Execute returns the full detail for a segment by UUID.
func (uc *GetSegmentUseCase) Execute(ctx context.Context, id uuid.UUID) (*dto.SegmentDetail, error) {
	seg, err := uc.segments.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return uc.assemble(ctx, seg)
}

// ExecuteByExternalID returns the full detail for a segment by external id.
func (uc *GetSegmentUseCase) ExecuteByExternalID(ctx context.Context, externalID string) (*dto.SegmentDetail, error) {
	seg, err := uc.segments.GetByExternalID(ctx, externalID)
	if err != nil {
		return nil, err
	}
	return uc.assemble(ctx, seg)
}

func (uc *GetSegmentUseCase) assemble(ctx context.Context, seg *model.StreetSegment) (*dto.SegmentDetail, error) {
	fixtures, err := uc.fixtures.ListBySegment(ctx, seg.ID)
	if err != nil {
		return nil, err
	}
	analyses, err := uc.analyses.ListBySegment(ctx, seg.ID)
	if err != nil {
		return nil, err
	}
	return &dto.SegmentDetail{Segment: seg, Fixtures: fixtures, Analyses: analyses}, nil
}
