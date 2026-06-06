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
	LightingDensity    float64     `json:"lighting_density"`
	RecommendedDensity float64     `json:"recommended_density"`
	Adequacy           float64     `json:"adequacy"`
	RiskScore          float64     `json:"risk_score"`
	RiskLevel          string      `json:"risk_level"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at"`
}
