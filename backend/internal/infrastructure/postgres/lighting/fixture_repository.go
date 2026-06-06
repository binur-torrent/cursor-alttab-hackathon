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

// FixtureRepo implements repository.FixtureRepository with PostgreSQL.
type FixtureRepo struct {
	db *pgxpool.Pool
}

// NewFixtureRepo creates a new FixtureRepo.
func NewFixtureRepo(db *pgxpool.Pool) *FixtureRepo {
	return &FixtureRepo{db: db}
}

func (r *FixtureRepo) CreateBatch(ctx context.Context, fixtures []*model.LightFixture) error {
	if len(fixtures) == 0 {
		return nil
	}
	batch := make([][]interface{}, 0, len(fixtures))
	now := time.Now().UTC()
	for _, f := range fixtures {
		if f.ID == uuid.Nil {
			f.ID = uuid.New()
		}
		if f.CreatedAt.IsZero() {
			f.CreatedAt = now
		}
		batch = append(batch, []interface{}{f.ID, f.SegmentID, f.Type, f.Lat, f.Lon, f.Confidence, f.Source, f.CreatedAt})
	}
	_, err := r.db.CopyFrom(ctx,
		[]string{"light_fixtures"},
		[]string{"id", "segment_id", "type", "lat", "lon", "confidence", "source", "created_at"},
		pgx.CopyFromRows(batch),
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to insert fixtures", err)
	}
	return nil
}

func (r *FixtureRepo) ListBySegment(ctx context.Context, segmentID uuid.UUID) ([]*model.LightFixture, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, segment_id, type, lat, lon, confidence, source, created_at
		 FROM light_fixtures WHERE segment_id=$1 ORDER BY created_at`, segmentID)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to list fixtures", err)
	}
	defer rows.Close()

	var fixtures []*model.LightFixture
	for rows.Next() {
		var f model.LightFixture
		if err := rows.Scan(&f.ID, &f.SegmentID, &f.Type, &f.Lat, &f.Lon, &f.Confidence, &f.Source, &f.CreatedAt); err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "failed to scan fixture", err)
		}
		fixtures = append(fixtures, &f)
	}
	return fixtures, nil
}

func (r *FixtureRepo) DeleteBySegment(ctx context.Context, segmentID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM light_fixtures WHERE segment_id=$1`, segmentID)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to delete fixtures", err)
	}
	return nil
}
