package usecase

import (
	"context"

	"github.com/masterfabric-go/masterfabric/internal/domain/lighting/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/lighting/repository"
)

// SimulateScenarioUseCase runs an adaptive-lighting policy simulation over the
// (optionally district-scoped) street network.
type SimulateScenarioUseCase struct {
	segments repository.SegmentRepository
}

// NewSimulateScenarioUseCase creates a new SimulateScenarioUseCase.
func NewSimulateScenarioUseCase(segments repository.SegmentRepository) *SimulateScenarioUseCase {
	return &SimulateScenarioUseCase{segments: segments}
}

// Execute loads the matching segments and computes the baseline-vs-proposed
// energy and safety outcome for the given policy parameters.
func (uc *SimulateScenarioUseCase) Execute(
	ctx context.Context,
	params model.ScenarioParams,
	filter repository.SegmentFilter,
) (*model.ScenarioResult, error) {
	segments, err := uc.segments.ListAll(ctx, filter)
	if err != nil {
		return nil, err
	}
	result := model.SimulateScenario(segments, params)
	return &result, nil
}
