package dto

import "github.com/masterfabric-go/masterfabric/internal/domain/lighting/model"

// AnalyzeRequest is the body for POST /lighting/analyze - analyze a single point
// in the city on demand (e.g. "analyze any address").
type AnalyzeRequest struct {
	Lat      float64 `json:"lat" validate:"required"`
	Lon      float64 `json:"lon" validate:"required"`
	Heading  float64 `json:"heading"`
	RoadType string  `json:"road_type"`
	LengthM  float64 `json:"length_m"`
	IsNight  bool    `json:"is_night"`
	Address  string  `json:"address"`
}

// AnalyzeResult is the response (and the AI worker's output shape). The worker
// returns detected counts + scene-composition features; the Go scoring model is
// the authority for the three scores (recomputed in the use case).
type AnalyzeResult struct {
	StreetLightCount int     `json:"street_light_count"`
	PoleCount        int     `json:"pole_count"`
	TreeCount        int     `json:"tree_count"`
	VegetationRatio  float64 `json:"vegetation_ratio"`
	BuildingRatio    float64 `json:"building_ratio"`
	SidewalkRatio    float64 `json:"sidewalk_ratio"`
	SkyRatio         float64 `json:"sky_ratio"`
	RoadWidthM       float64 `json:"road_width_m"`
	DetectorBackend  string  `json:"detector_backend"`
	FacesBlurred     int     `json:"faces_blurred"`
	PlatesBlurred    int     `json:"plates_blurred"`
	Anonymized       bool    `json:"anonymized"`

	Adequacy               float64 `json:"adequacy"`
	LightingDensity        float64 `json:"lighting_density"`
	LightingSufficiency    float64 `json:"lighting_sufficiency"`
	Occlusion              float64 `json:"occlusion"`
	InfrastructureAdequacy float64 `json:"infrastructure_adequacy"`
	OverallScore           float64 `json:"overall_score"`
	RiskScore              float64 `json:"risk_score"`
	RiskLevel              string  `json:"risk_level"`

	RoadType    string  `json:"road_type"`
	ImageBase64 *string `json:"image_base64,omitempty"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Address     string  `json:"address,omitempty"`
	Source      string  `json:"source"` // "ai-worker" or "heuristic-fallback"
}

// AnalyzeSegmentResult is returned by POST /lighting/segments/analyze: the
// persisted segment created/updated from a clicked point, plus the raw analysis.
type AnalyzeSegmentResult struct {
	Segment  *model.StreetSegment `json:"segment"`
	Analysis *AnalyzeResult       `json:"analysis"`
}
