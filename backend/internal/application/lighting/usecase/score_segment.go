package usecase

import "github.com/masterfabric-go/masterfabric/internal/domain/lighting/model"

// ScoreSegmentUseCase computes a lighting-risk breakdown for arbitrary inputs
// (used for live "what-if" scoring and by the live-analysis path). Pure: it
// delegates to the shared domain scoring model.
type ScoreSegmentUseCase struct{}

// NewScoreSegmentUseCase creates a new ScoreSegmentUseCase.
func NewScoreSegmentUseCase() *ScoreSegmentUseCase {
	return &ScoreSegmentUseCase{}
}

// Execute returns the risk breakdown for the given fixture count / road context.
func (uc *ScoreSegmentUseCase) Execute(streetlights int, lengthM float64, roadType string, nightRatio float64) model.RiskBreakdown {
	return model.ScoreSegment(streetlights, lengthM, roadType, nightRatio)
}
