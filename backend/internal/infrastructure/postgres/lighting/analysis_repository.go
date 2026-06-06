package lighting

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/masterfabric-go/masterfabric/internal/domain/lighting/model"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

// AnalysisRepo implements repository.AnalysisRepository with PostgreSQL.
type AnalysisRepo struct {
	db *pgxpool.Pool
}

// NewAnalysisRepo creates a new AnalysisRepo.
func NewAnalysisRepo(db *pgxpool.Pool) *AnalysisRepo {
	return &AnalysisRepo{db: db}
}

func (r *AnalysisRepo) CreateBatch(ctx context.Context, analyses []*model.LightingAnalysis) error {
	if len(analyses) == 0 {
		return nil
	}
	now := time.Now().UTC()
	rows := make([][]interface{}, 0, len(analyses))
	for _, a := range analyses {
		if a.ID == uuid.Nil {
			a.ID = uuid.New()
		}
		if a.CreatedAt.IsZero() {
			a.CreatedAt = now
		}
		rows = append(rows, []interface{}{
			a.ID, a.SegmentID, a.ExternalID, a.Lat, a.Lon, a.Heading, a.CapturedAt, a.TimeOfDay,
			a.RoadType, a.StreetLightCount, a.PoleCount, a.Anonymized, a.FacesBlurred, a.PlatesBlurred,
			a.Backend, a.CreatedAt,
		})
	}
	_, err := r.db.CopyFrom(ctx,
		[]string{"lighting_analyses"},
		[]string{"id", "segment_id", "external_id", "lat", "lon", "heading", "captured_at", "time_of_day",
			"road_type", "street_light_count", "pole_count", "anonymized", "faces_blurred", "plates_blurred",
			"backend", "created_at"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to insert analyses", err)
	}
	return nil
}

func (r *AnalysisRepo) ListBySegment(ctx context.Context, segmentID uuid.UUID) ([]*model.LightingAnalysis, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, segment_id, external_id, lat, lon, heading, captured_at, time_of_day, road_type,
		        street_light_count, pole_count, anonymized, faces_blurred, plates_blurred, backend, created_at
		 FROM lighting_analyses WHERE segment_id=$1 ORDER BY created_at`, segmentID)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to list analyses", err)
	}
	defer rows.Close()

	var analyses []*model.LightingAnalysis
	for rows.Next() {
		var a model.LightingAnalysis
		if err := rows.Scan(&a.ID, &a.SegmentID, &a.ExternalID, &a.Lat, &a.Lon, &a.Heading, &a.CapturedAt,
			&a.TimeOfDay, &a.RoadType, &a.StreetLightCount, &a.PoleCount, &a.Anonymized, &a.FacesBlurred,
			&a.PlatesBlurred, &a.Backend, &a.CreatedAt); err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "failed to scan analysis", err)
		}
		analyses = append(analyses, &a)
	}
	return analyses, nil
}

func (r *AnalysisRepo) DeleteBySegment(ctx context.Context, segmentID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM lighting_analyses WHERE segment_id=$1`, segmentID)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to delete analyses", err)
	}
	return nil
}
