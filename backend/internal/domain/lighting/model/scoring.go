package model

// Lighting adequacy / risk scoring. This is a 1:1 port of
// ai/pipeline/scoring.py so seed scores (computed in Python) and live scores
// (computed here in Go) always agree.
//
//	density     = streetlights / (length_m / 100)
//	recommended = RecommendedDensity[road_type]
//	adequacy    = clamp(density / recommended, 0, 1)
//	base_risk   = (1 - adequacy) * 100
//	risk        = clamp(base_risk * road_weight * night_weight, 0, 100)

// RecommendedDensity is the target streetlight fixtures per 100 m by road class.
var RecommendedDensity = map[string]float64{
	"highway":     4.0,
	"primary":     3.0,
	"secondary":   2.5,
	"residential": 1.8,
	"service":     1.2,
}

// RoadWeight scales how much an underlit road of each class contributes to risk.
var RoadWeight = map[string]float64{
	"highway":     1.25,
	"primary":     1.15,
	"secondary":   1.05,
	"residential": 0.95,
	"service":     0.85,
}

// DefaultRoadType is used when a segment's road type is unknown.
const DefaultRoadType = "secondary"

// RiskBreakdown is the full scoring output for a segment or point.
type RiskBreakdown struct {
	Density     float64 `json:"lighting_density"`
	Recommended float64 `json:"recommended_density"`
	Adequacy    float64 `json:"adequacy"`
	RiskScore   float64 `json:"risk_score"`
	RiskLevel   string  `json:"risk_level"`
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
	return float64(int(v*10+0.5)) / 10
}

func round3(v float64) float64 {
	return float64(int(v*1000+0.5)) / 1000
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

// RiskLevelFor buckets a numeric risk score.
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

// ScoreSegment computes the full risk breakdown for a segment.
func ScoreSegment(streetlights int, lengthM float64, roadType string, nightRatio float64) RiskBreakdown {
	density := LightingDensity(streetlights, lengthM)
	recommended := RecommendedDensityFor(roadType)
	adequacy := 1.0
	if recommended > 0 {
		adequacy = clamp(density/recommended, 0, 1)
	}
	baseRisk := (1.0 - adequacy) * 100.0
	roadWeight := RoadWeightFor(roadType)
	nightWeight := 1.0 + 0.20*clamp(nightRatio, 0, 1)*(1.0-adequacy)
	risk := clamp(baseRisk*roadWeight*nightWeight, 0, 100)

	return RiskBreakdown{
		Density:     round3(density),
		Recommended: recommended,
		Adequacy:    round3(adequacy),
		RiskScore:   round1(risk),
		RiskLevel:   RiskLevelFor(risk),
	}
}
