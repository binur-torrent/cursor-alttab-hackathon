package usecase

import (
	"context"
	"fmt"
	"hash/fnv"
	"log/slog"

	"github.com/masterfabric-go/masterfabric/internal/application/lighting/dto"
	"github.com/masterfabric-go/masterfabric/internal/domain/lighting/model"
)

// AnalyzerPort is the boundary to the AI worker (FastAPI / HF inference).
// Implemented by internal/infrastructure/aiworker.
type AnalyzerPort interface {
	AnalyzePoint(ctx context.Context, req dto.AnalyzeRequest) (*dto.AnalyzeResult, error)
}

// AnalyzeLiveUseCase analyzes an arbitrary point on demand. It prefers the AI
// worker (which fetches Street View, anonymizes, and runs the CV model) and
// falls back to a deterministic Go heuristic so the live demo always works.
type AnalyzeLiveUseCase struct {
	worker AnalyzerPort // may be nil if no worker is configured
	log    *slog.Logger
}

// NewAnalyzeLiveUseCase creates a new AnalyzeLiveUseCase. Pass worker=nil to run
// in fallback-only mode.
func NewAnalyzeLiveUseCase(worker AnalyzerPort, log *slog.Logger) *AnalyzeLiveUseCase {
	return &AnalyzeLiveUseCase{worker: worker, log: log}
}

// Execute analyzes the requested point.
func (uc *AnalyzeLiveUseCase) Execute(ctx context.Context, req dto.AnalyzeRequest) (*dto.AnalyzeResult, error) {
	if req.RoadType == "" {
		req.RoadType = model.DefaultRoadType
	}
	if req.LengthM <= 0 {
		req.LengthM = 100
	}

	if uc.worker != nil {
		result, err := uc.worker.AnalyzePoint(ctx, req)
		if err == nil && result != nil {
			result.Lat, result.Lon, result.Address = req.Lat, req.Lon, req.Address
			result.Source = "ai-worker"
			return result, nil
		}
		if uc.log != nil {
			uc.log.Warn("ai worker unavailable, using heuristic fallback", "error", err)
		}
	}

	return uc.heuristic(req), nil
}

// heuristic produces a deterministic, explainable estimate without the worker.
// It mirrors the Python heuristic detector and reuses the shared scoring model.
func (uc *AnalyzeLiveUseCase) heuristic(req dto.AnalyzeRequest) *dto.AnalyzeResult {
	h := fnv.New32a()
	_, _ = h.Write([]byte(fmt.Sprintf("%.5f,%.5f", req.Lat, req.Lon)))
	seed := h.Sum32()
	streetLights := int(seed % 4)      // 0..3 fixtures in frame
	poles := int((seed / 7) % 5)       // 0..4 poles

	nightRatio := 0.0
	if req.IsNight {
		nightRatio = 1.0
	}
	bd := model.ScoreSegment(streetLights, req.LengthM, req.RoadType, nightRatio)

	return &dto.AnalyzeResult{
		StreetLightCount: streetLights,
		PoleCount:        poles,
		DetectorBackend:  "heuristic-fallback",
		FacesBlurred:     0,
		PlatesBlurred:    0,
		Anonymized:       true,
		RiskScore:        bd.RiskScore,
		RiskLevel:        bd.RiskLevel,
		Adequacy:         bd.Adequacy,
		LightingDensity:  bd.Density,
		Lat:              req.Lat,
		Lon:              req.Lon,
		Address:          req.Address,
		Source:           "heuristic-fallback",
	}
}
