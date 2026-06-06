// Package model contains the lighting bounded context domain entities.
// Following masterfabric-go clean architecture, this layer has zero external
// dependencies beyond uuid + stdlib.
package model

import (
	"time"

	"github.com/google/uuid"
)

// RoadType classifies a street segment for lighting requirements.
type RoadType string

const (
	RoadHighway     RoadType = "highway"
	RoadPrimary     RoadType = "primary"
	RoadSecondary   RoadType = "secondary"
	RoadResidential RoadType = "residential"
	RoadService     RoadType = "service"
)

// RiskLevel is a coarse bucket derived from the numeric risk score.
type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

// StreetSegment is the core digital-twin entity: a ~150-200 m stretch of street
// with aggregated streetlight detections and a computed lighting-risk score.
type StreetSegment struct {
	ID                 uuid.UUID   `json:"id"`
	ExternalID         string      `json:"external_id"`
	Name               string      `json:"name"`
	District           string      `json:"district"`
	RoadType           string      `json:"road_type"`
	CentroidLat        float64     `json:"centroid_lat"`
	CentroidLon        float64     `json:"centroid_lon"`
	Geometry           [][]float64 `json:"geometry"` // polyline [[lat,lon],...]
	LengthM            float64     `json:"length_m"`
	SampleCount        int         `json:"sample_count"`
	StreetLightCount   int         `json:"street_light_count"`
	PoleCount          int         `json:"pole_count"`
	NightSampleRatio   float64     `json:"night_sample_ratio"`

	// Environmental features detected from imagery that affect illumination.
	// These drive the occlusion / infrastructure scores and the recommendation
	// engine. Defaults (zero) are handled gracefully by the scoring model.
	TreeCount       int     `json:"tree_count"`
	VegetationRatio float64 `json:"vegetation_ratio"` // 0..1 canopy/greenery in view
	BuildingRatio   float64 `json:"building_ratio"`   // 0..1 facade coverage in view
	RoadWidthM      float64 `json:"road_width_m"`     // estimated carriageway width
	SidewalkRatio   float64 `json:"sidewalk_ratio"`   // 0..1 sidewalk presence/coverage
	SkyRatio        float64 `json:"sky_ratio"`        // 0..1 open-sky visibility
	BrightnessFactor float64 `json:"brightness_factor"` // 1.0 = nominal output

	LightingDensity    float64 `json:"lighting_density"`
	RecommendedDensity float64 `json:"recommended_density"`
	Adequacy           float64 `json:"adequacy"`

	// The three headline 0..100 scores plus the composite.
	LightingSufficiency    float64 `json:"lighting_sufficiency"`    // higher = better
	Occlusion              float64 `json:"occlusion"`               // higher = more blocked (worse)
	InfrastructureAdequacy float64 `json:"infrastructure_adequacy"` // higher = better
	OverallScore           float64 `json:"overall_score"`           // higher = better

	RiskScore float64   `json:"risk_score"` // 100 - overall_score
	RiskLevel string    `json:"risk_level"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
