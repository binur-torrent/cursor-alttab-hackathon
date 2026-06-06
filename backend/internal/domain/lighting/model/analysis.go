package model

import (
	"time"

	"github.com/google/uuid"
)

// LightingAnalysis is a single analyzed street-level frame: the per-image record
// of what the CV pipeline detected, with KVKK anonymization provenance.
type LightingAnalysis struct {
	ID               uuid.UUID  `json:"id"`
	SegmentID        uuid.UUID  `json:"segment_id"`
	ExternalID       string     `json:"external_id"` // source frame id (e.g. Mapillary id)
	Lat              float64    `json:"lat"`
	Lon              float64    `json:"lon"`
	Heading          float64    `json:"heading"`
	CapturedAt       *time.Time `json:"captured_at,omitempty"`
	TimeOfDay        string     `json:"time_of_day"`
	RoadType         string     `json:"road_type"`
	StreetLightCount int        `json:"street_light_count"`
	PoleCount        int        `json:"pole_count"`
	Anonymized       bool       `json:"anonymized"`
	FacesBlurred     int        `json:"faces_blurred"`
	PlatesBlurred    int        `json:"plates_blurred"`
	Backend          string     `json:"backend"`
	CreatedAt        time.Time  `json:"created_at"`
}
