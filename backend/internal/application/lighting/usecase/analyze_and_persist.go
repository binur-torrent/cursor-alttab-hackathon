package usecase

import (
	"context"
	"fmt"
	"math"

	"github.com/masterfabric-go/masterfabric/internal/application/lighting/dto"
	"github.com/masterfabric-go/masterfabric/internal/domain/lighting/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/lighting/repository"
)

// AnalyzeAndPersistUseCase analyzes an arbitrary clicked point (via the AI
// worker + Street View, or the heuristic fallback) and persists the result as a
// street segment so it appears on the map and survives a reload.
type AnalyzeAndPersistUseCase struct {
	analyze  *AnalyzeLiveUseCase
	segments repository.SegmentRepository
}

// NewAnalyzeAndPersistUseCase creates a new AnalyzeAndPersistUseCase.
func NewAnalyzeAndPersistUseCase(analyze *AnalyzeLiveUseCase, segments repository.SegmentRepository) *AnalyzeAndPersistUseCase {
	return &AnalyzeAndPersistUseCase{analyze: analyze, segments: segments}
}

// Execute runs analysis and upserts a segment keyed by rounded coordinates
// (re-analyzing the same spot updates it rather than duplicating).
func (uc *AnalyzeAndPersistUseCase) Execute(ctx context.Context, req dto.AnalyzeRequest) (*dto.AnalyzeSegmentResult, error) {
	if req.RoadType == "" {
		req.RoadType = model.DefaultRoadType
	}
	if req.LengthM <= 0 {
		req.LengthM = 100
	}

	res, err := uc.analyze.Execute(ctx, req)
	if err != nil {
		return nil, err
	}

	externalID := fmt.Sprintf("live-%.5f-%.5f", req.Lat, req.Lon)
	name := req.Address
	if name == "" {
		name = fmt.Sprintf("Analyzed street (%.4f, %.4f)", req.Lat, req.Lon)
	}

	nightRatio := 0.0
	if req.IsNight {
		nightRatio = 1.0
	}

	seg := &model.StreetSegment{
		ExternalID:       externalID,
		Name:             name,
		District:         "Live analysis",
		RoadType:         req.RoadType,
		CentroidLat:      req.Lat,
		CentroidLon:      req.Lon,
		Geometry:         lineAround(req.Lat, req.Lon, req.LengthM),
		LengthM:          req.LengthM,
		SampleCount:      1,
		StreetLightCount: res.StreetLightCount,
		PoleCount:        res.PoleCount,
		NightSampleRatio: nightRatio,
		TreeCount:        res.TreeCount,
		VegetationRatio:  res.VegetationRatio,
		BuildingRatio:    res.BuildingRatio,
		RoadWidthM:       res.RoadWidthM,
		SidewalkRatio:    res.SidewalkRatio,
		SkyRatio:         res.SkyRatio,
		BrightnessFactor: 1,
	}
	model.ApplyScores(seg, model.ScoreEnv(model.FeaturesOf(seg)))

	// Preserve identity/created_at if this point was analyzed before.
	if existing, err := uc.segments.GetByExternalID(ctx, externalID); err == nil && existing != nil {
		seg.ID = existing.ID
		seg.CreatedAt = existing.CreatedAt
	}
	if err := uc.segments.Upsert(ctx, seg); err != nil {
		return nil, err
	}

	return &dto.AnalyzeSegmentResult{Segment: seg, Analysis: res}, nil
}

// lineAround builds a short east-west polyline (~lengthM) centred on a point so
// the analyzed segment renders on the map.
func lineAround(lat, lon, lengthM float64) [][]float64 {
	half := lengthM / 2.0
	metersPerDegLon := 111320.0 * math.Cos(lat*math.Pi/180.0)
	if metersPerDegLon < 1 {
		metersPerDegLon = 111320.0
	}
	dLon := half / metersPerDegLon
	pts := [][]float64{}
	const steps = 4
	for i := 0; i <= steps; i++ {
		t := float64(i)/float64(steps)*2 - 1 // -1 .. 1
		pts = append(pts, []float64{lat, lon + dLon*t})
	}
	return pts
}
