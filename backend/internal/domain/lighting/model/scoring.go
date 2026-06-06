package model

import "math"

// Lighting assessment scoring (v2). This is a 1:1 port of
// ai/pipeline/scoring.py so seed scores (computed in Python) and live scores
// (computed here in Go) always agree. See .cursor/rules/scoring-consistency.mdc.
//
// From environmental features detected in street-level imagery we derive three
// headline 0..100 scores and a composite "overall" lighting score:
//
//	adequacy   = clamp(density * brightness / recommended, 0, 1)
//	LSS        = effective_adequacy * 100 * night_factor   (lamp coverage)
//	OCC        = vegetation + tree + building blocking      (higher = worse)
//	IAS        = adequacy + sidewalk + pole support         (physical infra)
//	overall    = 0.50*LSS + 0.30*(100 - OCC) + 0.20*IAS
//	risk_score = 100 - overall
//
// The model is deliberately simple and explainable for the demo: every score
// is a transparent function of detected, urban-only features (never people).

// RecommendedDensity is the target streetlight fixtures per 100 m by road class.
var RecommendedDensity = map[string]float64{
	"highway":     4.0,
	"primary":     3.0,
	"secondary":   2.5,
	"residential": 1.8,
	"service":     1.2,
}

// RoadWeight scales how much an underlit road of each class contributes to risk.
// Retained for compatibility with the network-simulation energy model.
var RoadWeight = map[string]float64{
	"highway":     1.25,
	"primary":     1.15,
	"secondary":   1.05,
	"residential": 0.95,
	"service":     0.85,
}

// RoadWidth is the typical carriageway width (m) by road class. Wider roads need
// proportionally more light to cover, so they demand higher density.
var RoadWidth = map[string]float64{
	"highway":     20.0,
	"primary":     16.0,
	"secondary":   12.0,
	"residential": 8.0,
	"service":     6.0,
}

// DefaultRoadType is used when a segment's road type is unknown.
const DefaultRoadType = "secondary"

// Scoring blend weights for the composite overall score.
const (
	weightSufficiency = 0.50
	weightClearness   = 0.30 // applied to (100 - occlusion)
	weightInfra       = 0.20
)

// Features is the full input to the scoring model: detected urban assets plus
// environmental context. All ratios are 0..1; zero values are handled safely.
type Features struct {
	Streetlights     int
	PoleCount        int
	LengthM          float64
	RoadType         string
	NightRatio       float64
	RoadWidthM       float64
	TreeCount        int
	VegetationRatio  float64
	BuildingRatio    float64
	SidewalkRatio    float64
	SkyRatio         float64
	BrightnessFactor float64 // 1.0 = nominal; >1 brighter lamps, <1 dimmed
}

// RiskBreakdown is the full scoring output for a segment or point.
type RiskBreakdown struct {
	Density                float64 `json:"lighting_density"`
	Recommended            float64 `json:"recommended_density"`
	Adequacy               float64 `json:"adequacy"`
	LightingSufficiency    float64 `json:"lighting_sufficiency"`
	Occlusion              float64 `json:"occlusion"`
	InfrastructureAdequacy float64 `json:"infrastructure_adequacy"`
	OverallScore           float64 `json:"overall_score"`
	RiskScore              float64 `json:"risk_score"`
	RiskLevel              string  `json:"risk_level"`
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

func round3(v float64) float64 {
	return math.Round(v*1000) / 1000
}

// LightingDensity returns fixtures per 100 m of segment length.
func LightingDensity(streetlights int, lengthM float64) float64 {
	if lengthM <= 0 {
		return 0
	}
	return float64(streetlights) / (lengthM / 100.0)
}

// RecommendedDensityFor returns the target density for a road type.
func RecommendedDensityFor(roadType string) float64 {
	if v, ok := RecommendedDensity[roadType]; ok {
		return v
	}
	return RecommendedDensity[DefaultRoadType]
}

// RoadWeightFor returns the risk weight for a road type.
func RoadWeightFor(roadType string) float64 {
	if v, ok := RoadWeight[roadType]; ok {
		return v
	}
	return RoadWeight[DefaultRoadType]
}

// RoadWidthFor returns the typical carriageway width for a road type.
func RoadWidthFor(roadType string) float64 {
	if v, ok := RoadWidth[roadType]; ok {
		return v
	}
	return RoadWidth[DefaultRoadType]
}

// RiskLevelFor buckets a numeric risk score (100 - overall lighting score).
func RiskLevelFor(score float64) string {
	switch {
	case score >= 75:
		return string(RiskCritical)
	case score >= 50:
		return string(RiskHigh)
	case score >= 25:
		return string(RiskMedium)
	default:
		return string(RiskLow)
	}
}

// ScoreEnv computes the full assessment from environmental features. Pure and
// deterministic - the single source of truth shared with the Python pipeline.
func ScoreEnv(f Features) RiskBreakdown {
	if f.RoadType == "" {
		f.RoadType = DefaultRoadType
	}
	if f.LengthM <= 0 {
		f.LengthM = 100
	}
	brightness := f.BrightnessFactor
	if brightness <= 0 {
		brightness = 1.0
	}

	density := LightingDensity(f.Streetlights, f.LengthM)
	recommended := RecommendedDensityFor(f.RoadType)
	adequacy := 1.0
	if recommended > 0 {
		adequacy = clamp(density*brightness/recommended, 0, 1)
	}

	// Lighting Sufficiency: lamp coverage, penalised on wide roads and where a
	// segment is mostly observed at night yet still underlit.
	nightFactor := 1.0 - 0.15*clamp(f.NightRatio, 0, 1)*(1.0-adequacy)
	baseWidth := RoadWidthFor(f.RoadType)
	roadWidth := f.RoadWidthM
	if roadWidth <= 0 {
		roadWidth = baseWidth
	}
	widthDemand := clamp(roadWidth/baseWidth, 0.7, 1.8)
	effectiveAdequacy := clamp(adequacy/widthDemand, 0, 1)
	lss := clamp(effectiveAdequacy*100*nightFactor, 0, 100)

	// Occlusion: how much vegetation and built mass block lamp output.
	occRaw := 0.55*clamp(f.VegetationRatio, 0, 1) +
		0.25*clamp(float64(f.TreeCount)/8.0, 0, 1) +
		0.20*clamp(f.BuildingRatio, 0, 1)
	occ := clamp(occRaw, 0, 1) * 100

	// Infrastructure Adequacy: physical readiness (poles, sidewalks, coverage).
	expectedPoles := math.Ceil(recommended * (f.LengthM / 100.0))
	poleSupport := 0.0
	if expectedPoles > 0 {
		poleSupport = clamp(float64(f.PoleCount)/expectedPoles, 0, 1)
	}
	iasRaw := 0.55*adequacy + 0.25*clamp(f.SidewalkRatio, 0, 1) + 0.20*poleSupport
	ias := clamp(iasRaw, 0, 1) * 100

	overall := clamp(weightSufficiency*lss+weightClearness*(100-occ)+weightInfra*ias, 0, 100)
	risk := clamp(100-overall, 0, 100)

	return RiskBreakdown{
		Density:                round3(density),
		Recommended:            recommended,
		Adequacy:               round3(adequacy),
		LightingSufficiency:    round1(lss),
		Occlusion:              round1(occ),
		InfrastructureAdequacy: round1(ias),
		OverallScore:           round1(overall),
		RiskScore:              round1(risk),
		RiskLevel:              RiskLevelFor(risk),
	}
}

// ScoreSegment computes the breakdown from lamp count + road context only
// (no environmental occlusion features). Retained for the network simulation
// and the heuristic fallback path.
func ScoreSegment(streetlights int, lengthM float64, roadType string, nightRatio float64) RiskBreakdown {
	return ScoreEnv(Features{
		Streetlights: streetlights,
		LengthM:      lengthM,
		RoadType:     roadType,
		NightRatio:   nightRatio,
	})
}
