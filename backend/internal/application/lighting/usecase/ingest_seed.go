package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/masterfabric-go/masterfabric/internal/application/lighting/dto"
	"github.com/masterfabric-go/masterfabric/internal/domain/lighting/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/lighting/repository"
)

// IngestSeedUseCase loads precomputed, anonymized AI pipeline output (seed JSON)
// into the lighting tables. Idempotent: segments are upserted by external_id and
// their fixtures/analyses are replaced.
type IngestSeedUseCase struct {
	segments repository.SegmentRepository
	fixtures repository.FixtureRepository
	analyses repository.AnalysisRepository
}

// NewIngestSeedUseCase creates a new IngestSeedUseCase.
func NewIngestSeedUseCase(
	segments repository.SegmentRepository,
	fixtures repository.FixtureRepository,
	analyses repository.AnalysisRepository,
) *IngestSeedUseCase {
	return &IngestSeedUseCase{segments: segments, fixtures: fixtures, analyses: analyses}
}

// seedFile mirrors the JSON emitted by ai/pipeline (run.py / generate_seed.py).
type seedFile struct {
	SourceDataset   string        `json:"source_dataset"`
	DetectorBackend string        `json:"detector_backend"`
	SegmentCount    int           `json:"segment_count"`
	Segments        []seedSegment `json:"segments"`
}

type seedSegment struct {
	ExternalID         string       `json:"external_id"`
	Name               string       `json:"name"`
	District           string       `json:"district"`
	RoadType           string       `json:"road_type"`
	CentroidLat        float64      `json:"centroid_lat"`
	CentroidLon        float64      `json:"centroid_lon"`
	Geometry           [][]float64  `json:"geometry"`
	LengthM            float64      `json:"length_m"`
	SampleCount        int          `json:"sample_count"`
	StreetLightCount   int          `json:"street_light_count"`
	PoleCount          int          `json:"pole_count"`
	NightSampleRatio   float64      `json:"night_sample_ratio"`
	TreeCount          int          `json:"tree_count"`
	VegetationRatio    float64      `json:"vegetation_ratio"`
	BuildingRatio      float64      `json:"building_ratio"`
	RoadWidthM         float64      `json:"road_width_m"`
	SidewalkRatio      float64      `json:"sidewalk_ratio"`
	SkyRatio           float64      `json:"sky_ratio"`
	BrightnessFactor   float64      `json:"brightness_factor"`
	LightingDensity    float64      `json:"lighting_density"`
	RecommendedDensity float64      `json:"recommended_density"`
	Adequacy           float64      `json:"adequacy"`

	LightingSufficiency    float64 `json:"lighting_sufficiency"`
	Occlusion              float64 `json:"occlusion"`
	InfrastructureAdequacy float64 `json:"infrastructure_adequacy"`
	OverallScore           float64 `json:"overall_score"`

	RiskScore float64      `json:"risk_score"`
	RiskLevel string       `json:"risk_level"`
	Samples   []seedSample `json:"samples"`
}

type seedSample struct {
	ID               string  `json:"id"`
	Lat              float64 `json:"lat"`
	Lon              float64 `json:"lon"`
	Heading          float64 `json:"heading"`
	CapturedAt       string  `json:"captured_at"`
	TimeOfDay        string  `json:"time_of_day"`
	RoadType         string  `json:"road_type"`
	StreetLightCount int     `json:"street_light_count"`
	PoleCount        int     `json:"pole_count"`
	Anonymized       bool    `json:"anonymized"`
	FacesBlurred     int     `json:"faces_blurred"`
	PlatesBlurred    int     `json:"plates_blurred"`
	Backend          string  `json:"backend"`
}

// Execute parses and ingests the seed bytes. Returns counts of what was loaded.
func (uc *IngestSeedUseCase) Execute(ctx context.Context, raw []byte) (*dto.IngestResult, error) {
	var file seedFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("invalid seed json: %w", err)
	}

	result := &dto.IngestResult{}
	for i := range file.Segments {
		s := &file.Segments[i]
		seg := &model.StreetSegment{
			ExternalID:         s.ExternalID,
			Name:               s.Name,
			District:           s.District,
			RoadType:           s.RoadType,
			CentroidLat:        s.CentroidLat,
			CentroidLon:        s.CentroidLon,
			Geometry:           s.Geometry,
			LengthM:            s.LengthM,
			SampleCount:        s.SampleCount,
			StreetLightCount:   s.StreetLightCount,
			PoleCount:          s.PoleCount,
			NightSampleRatio:   s.NightSampleRatio,
			TreeCount:          s.TreeCount,
			VegetationRatio:    s.VegetationRatio,
			BuildingRatio:      s.BuildingRatio,
			RoadWidthM:         s.RoadWidthM,
			SidewalkRatio:      s.SidewalkRatio,
			SkyRatio:           s.SkyRatio,
			BrightnessFactor:   s.BrightnessFactor,
			LightingDensity:    s.LightingDensity,
			RecommendedDensity: s.RecommendedDensity,
			Adequacy:           s.Adequacy,

			LightingSufficiency:    s.LightingSufficiency,
			Occlusion:              s.Occlusion,
			InfrastructureAdequacy: s.InfrastructureAdequacy,
			OverallScore:           s.OverallScore,

			RiskScore: s.RiskScore,
			RiskLevel: s.RiskLevel,
		}
		// Backfill scores if the seed predates scoring v2 (older JSON files).
		if seg.OverallScore == 0 && seg.RiskScore == 0 {
			model.ApplyScores(seg, model.ScoreEnv(model.FeaturesOf(seg)))
		}
		if seg.BrightnessFactor == 0 {
			seg.BrightnessFactor = 1
		}
		if err := uc.segments.Upsert(ctx, seg); err != nil {
			return nil, err
		}
		result.Segments++

		// Re-seed children for this segment (idempotent).
		if err := uc.fixtures.DeleteBySegment(ctx, seg.ID); err != nil {
			return nil, err
		}
		if err := uc.analyses.DeleteBySegment(ctx, seg.ID); err != nil {
			return nil, err
		}

		var fixtures []*model.LightFixture
		var analyses []*model.LightingAnalysis
		for _, sample := range s.Samples {
			capturedAt := parseTime(sample.CapturedAt)
			analyses = append(analyses, &model.LightingAnalysis{
				SegmentID:        seg.ID,
				ExternalID:       sample.ID,
				Lat:              sample.Lat,
				Lon:              sample.Lon,
				Heading:          sample.Heading,
				CapturedAt:       capturedAt,
				TimeOfDay:        sample.TimeOfDay,
				RoadType:         sample.RoadType,
				StreetLightCount: sample.StreetLightCount,
				PoleCount:        sample.PoleCount,
				Anonymized:       sample.Anonymized,
				FacesBlurred:     sample.FacesBlurred,
				PlatesBlurred:    sample.PlatesBlurred,
				Backend:          sample.Backend,
			})
			for j := 0; j < sample.StreetLightCount; j++ {
				fixtures = append(fixtures, &model.LightFixture{
					SegmentID:  seg.ID,
					Type:       string(model.FixtureStreetLight),
					Lat:        sample.Lat,
					Lon:        sample.Lon,
					Confidence: 0.7,
					Source:     sample.Backend,
				})
			}
			for j := 0; j < sample.PoleCount; j++ {
				fixtures = append(fixtures, &model.LightFixture{
					SegmentID:  seg.ID,
					Type:       string(model.FixturePole),
					Lat:        sample.Lat,
					Lon:        sample.Lon,
					Confidence: 0.6,
					Source:     sample.Backend,
				})
			}
		}
		if err := uc.fixtures.CreateBatch(ctx, fixtures); err != nil {
			return nil, err
		}
		if err := uc.analyses.CreateBatch(ctx, analyses); err != nil {
			return nil, err
		}
		result.Fixtures += len(fixtures)
		result.Analyses += len(analyses)
	}
	return result, nil
}

func parseTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}
