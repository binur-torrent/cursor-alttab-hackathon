package dto

import "github.com/masterfabric-go/masterfabric/internal/domain/lighting/model"

// RescoreRequest is the body for POST /lighting/segments/{id}/rescore. It
// describes a what-if intervention on a single segment. With all-zero values it
// returns the current scores plus recommendations (no change). When Persist is
// true the new state is written back so the map recolors.
type RescoreRequest struct {
	AddedLamps       int     `json:"added_lamps"`       // lamp posts to install (>=0)
	TrimVegetation   bool    `json:"trim_vegetation"`   // prune canopy occlusion
	BrightnessFactor float64 `json:"brightness_factor"` // 1.0 nominal; >1 brighter
	Persist          bool    `json:"persist"`           // write the intervention back
}

// RescoreResult returns the baseline vs projected breakdown, the recommendation
// set for the current state, and (when persisted) the updated segment.
type RescoreResult struct {
	SegmentID       string                 `json:"segment_id"`
	ExternalID      string                 `json:"external_id"`
	Baseline        model.RiskBreakdown    `json:"baseline"`
	Projected       model.RiskBreakdown    `json:"projected"`
	Recommendations []model.Recommendation `json:"recommendations"`
	Applied         bool                   `json:"applied"`
	Segment         *model.StreetSegment   `json:"segment"`
}
