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
			// The Go scoring model is the single source of truth: recompute the
			// scores from the worker's detected features so live + seed agree.
			uc.applyScores(result, req)
			return result, nil
		}
		if uc.log != nil {
			uc.log.Warn("ai worker unavailable, using heuristic fallback", "error", err)
		}
	}

	return uc.heuristic(req), nil
}

// applyScores fills the v2 scores on a result from its detected features.
func (uc *AnalyzeLiveUseCase) applyScores(r *dto.AnalyzeResult, req dto.AnalyzeRequest) {
	nightRatio := 0.0
	if req.IsNight {
		nightRatio = 1.0
	}
	width := r.RoadWidthM
	if width <= 0 {
		width = model.RoadWidthFor(req.RoadType)
	}
	bd := model.ScoreEnv(model.Features{
		Streetlights:    r.StreetLightCount,
		PoleCount:       r.PoleCount,
		LengthM:         req.LengthM,
		RoadType:        req.RoadType,
		NightRatio:      nightRatio,
		RoadWidthM:      width,
		TreeCount:       r.TreeCount,
		VegetationRatio: r.VegetationRatio,
		BuildingRatio:   r.BuildingRatio,
		SidewalkRatio:   r.SidewalkRatio,
		SkyRatio:        r.SkyRatio,
	})
	r.RoadType = req.RoadType
	r.RoadWidthM = width
	r.Adequacy = bd.Adequacy
	r.LightingDensity = bd.Density
	r.LightingSufficiency = bd.LightingSufficiency
	r.Occlusion = bd.Occlusion
	r.InfrastructureAdequacy = bd.InfrastructureAdequacy
	r.OverallScore = bd.OverallScore
	r.RiskScore = bd.RiskScore
	r.RiskLevel = bd.RiskLevel
}

// heuristic produces a deterministic, explainable estimate without the worker.
// It mirrors the Python heuristic detector and reuses the shared scoring model.
func (uc *AnalyzeLiveUseCase) heuristic(req dto.AnalyzeRequest) *dto.AnalyzeResult {
	h := fnv.New32a()
	_, _ = h.Write([]byte(fmt.Sprintf("%.5f,%.5f", req.Lat, req.Lon)))
	seed := h.Sum32()
	streetLights := int(seed % 4) // 0..3 fixtures in frame
	poles := int((seed/7)%5) + streetLights
	veg := float64((seed/11)%65) / 100.0
	building := 0.15 + float64((seed/13)%55)/100.0
	sidewalk := float64((seed/17)%85) / 100.0
	sky := 1.0 - veg*0.6 - building*0.4

	result := &dto.AnalyzeResult{
		StreetLightCount: streetLights,
		PoleCount:        poles,
		TreeCount:        int(veg*12 + 0.5),
		VegetationRatio:  veg,
		BuildingRatio:    building,
		SidewalkRatio:    sidewalk,
		SkyRatio:         sky,
		RoadWidthM:       model.RoadWidthFor(req.RoadType),
		DetectorBackend:  "heuristic-fallback",
		Anonymized:       true,
		Lat:              req.Lat,
		Lon:              req.Lon,
		Address:          req.Address,
		Source:           "heuristic-fallback",
	}
	uc.applyScores(result, req)
	return result
}
