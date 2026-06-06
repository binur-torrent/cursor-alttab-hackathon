package usecase

import (
	"context"
	"math"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/application/lighting/dto"
	"github.com/masterfabric-go/masterfabric/internal/domain/lighting/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/lighting/repository"
)

// RescoreSegmentUseCase applies a what-if intervention to a single segment and
// returns the projected scores. It optionally persists the new state so the map
// and stats reflect the change ("install lamps -> the rate goes up").
type RescoreSegmentUseCase struct {
	segments repository.SegmentRepository
}

// NewRescoreSegmentUseCase creates a new RescoreSegmentUseCase.
func NewRescoreSegmentUseCase(segments repository.SegmentRepository) *RescoreSegmentUseCase {
	return &RescoreSegmentUseCase{segments: segments}
}

// Execute resolves the segment (by UUID or external id), computes baseline and
// projected breakdowns for the requested intervention, and persists when asked.
func (uc *RescoreSegmentUseCase) Execute(ctx context.Context, idParam string, req dto.RescoreRequest) (*dto.RescoreResult, error) {
	seg, err := uc.resolve(ctx, idParam)
	if err != nil {
		return nil, err
	}

	baseline := model.ScoreEnv(model.FeaturesOf(seg))
	recommendations := model.Recommend(seg)

	// Build the projected feature set from the intervention.
	f := model.FeaturesOf(seg)
	if req.AddedLamps > 0 {
		f.Streetlights += req.AddedLamps
		f.PoleCount += req.AddedLamps
	}
	if req.TrimVegetation {
		f.VegetationRatio = clamp01(seg.VegetationRatio * 0.4)
		f.TreeCount = int(math.Round(float64(seg.TreeCount) * 0.5))
	}
	if req.BrightnessFactor > 0 {
		f.BrightnessFactor = req.BrightnessFactor
	}
	projected := model.ScoreEnv(f)

	result := &dto.RescoreResult{
		SegmentID:       seg.ID.String(),
		ExternalID:      seg.ExternalID,
		Baseline:        baseline,
		Projected:       projected,
		Recommendations: recommendations,
		Segment:         seg,
	}

	if req.Persist {
		seg.StreetLightCount = f.Streetlights
		seg.PoleCount = f.PoleCount
		seg.VegetationRatio = f.VegetationRatio
		seg.TreeCount = f.TreeCount
		seg.BrightnessFactor = f.BrightnessFactor
		model.ApplyScores(seg, projected)
		if err := uc.segments.Upsert(ctx, seg); err != nil {
			return nil, err
		}
		result.Applied = true
		result.Segment = seg
	}

	return result, nil
}

func (uc *RescoreSegmentUseCase) resolve(ctx context.Context, idParam string) (*model.StreetSegment, error) {
	if id, err := uuid.Parse(idParam); err == nil {
		return uc.segments.GetByID(ctx, id)
	}
	return uc.segments.GetByExternalID(ctx, idParam)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
