package model

import (
	"time"

	"github.com/google/uuid"
)

// FixtureType is the kind of urban lighting asset detected.
type FixtureType string

const (
	FixtureStreetLight FixtureType = "street_light"
	FixturePole        FixtureType = "pole"
	FixtureUtilityPole FixtureType = "utility_pole"
)

// LightFixture is a single detected urban lighting asset, positioned at the
// frame it was detected in. Populated from per-frame detections during ingestion.
type LightFixture struct {
	ID         uuid.UUID `json:"id"`
	SegmentID  uuid.UUID `json:"segment_id"`
	Type       string    `json:"type"`
	Lat        float64   `json:"lat"`
	Lon        float64   `json:"lon"`
	Confidence float64   `json:"confidence"`
	Source     string    `json:"source"` // detector backend, e.g. "mask2former"
	CreatedAt  time.Time `json:"created_at"`
}
