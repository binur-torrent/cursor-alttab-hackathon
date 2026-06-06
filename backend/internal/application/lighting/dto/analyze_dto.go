package dto

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

// AnalyzeResult is the response (and the AI worker's output shape).
type AnalyzeResult struct {
	StreetLightCount int     `json:"street_light_count"`
	PoleCount        int     `json:"pole_count"`
	DetectorBackend  string  `json:"detector_backend"`
	FacesBlurred     int     `json:"faces_blurred"`
	PlatesBlurred    int     `json:"plates_blurred"`
	Anonymized       bool    `json:"anonymized"`
	RiskScore        float64 `json:"risk_score"`
	RiskLevel        string  `json:"risk_level"`
	Adequacy         float64 `json:"adequacy"`
	LightingDensity  float64 `json:"lighting_density"`
	ImageBase64      *string `json:"image_base64,omitempty"`
	Lat              float64 `json:"lat"`
	Lon              float64 `json:"lon"`
	Address          string  `json:"address,omitempty"`
	Source           string  `json:"source"` // "ai-worker" or "heuristic-fallback"
}
