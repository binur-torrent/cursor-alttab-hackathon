// Package repository defines persistence interfaces for the lighting context.
// Implementations live in internal/infrastructure/postgres/lighting.
package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/domain/lighting/model"
)

// SegmentFilter narrows a segment listing.
type SegmentFilter struct {
	District  string
	RoadType  string
	RiskLevel string
	MinRisk   float64
}

// SegmentRepository persists and queries street segments.
type SegmentRepository interface {
	Upsert(ctx context.Context, seg *model.StreetSegment) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.StreetSegment, error)
	GetByExternalID(ctx context.Context, externalID string) (*model.StreetSegment, error)
	List(ctx context.Context, filter SegmentFilter, offset, limit int) ([]*model.StreetSegment, int, error)
	ListAll(ctx context.Context, filter SegmentFilter) ([]*model.StreetSegment, error)
	Count(ctx context.Context) (int, error)
	Stats(ctx context.Context) (*model.CityStats, error)
}

// FixtureRepository persists and queries detected light fixtures.
type FixtureRepository interface {
	CreateBatch(ctx context.Context, fixtures []*model.LightFixture) error
	ListBySegment(ctx context.Context, segmentID uuid.UUID) ([]*model.LightFixture, error)
	DeleteBySegment(ctx context.Context, segmentID uuid.UUID) error
}

// AnalysisRepository persists and queries per-frame analyses.
type AnalysisRepository interface {
	CreateBatch(ctx context.Context, analyses []*model.LightingAnalysis) error
	ListBySegment(ctx context.Context, segmentID uuid.UUID) ([]*model.LightingAnalysis, error)
	DeleteBySegment(ctx context.Context, segmentID uuid.UUID) error
}
