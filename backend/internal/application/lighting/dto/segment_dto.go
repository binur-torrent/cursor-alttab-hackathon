// Package dto holds request/response shapes for the lighting context.
package dto

import "github.com/masterfabric-go/masterfabric/internal/domain/lighting/model"

// SegmentDetail is a single segment with its detected fixtures and the
// anonymized per-frame analyses behind its score.
type SegmentDetail struct {
	Segment  *model.StreetSegment      `json:"segment"`
	Fixtures []*model.LightFixture     `json:"fixtures"`
	Analyses []*model.LightingAnalysis `json:"analyses"`
}

// IngestResult summarizes a seed ingestion run.
type IngestResult struct {
	Segments int `json:"segments"`
	Fixtures int `json:"fixtures"`
	Analyses int `json:"analyses"`
}
