// Package lighting contains PostgreSQL implementations of the lighting domain
// repositories, mirroring the pgx patterns used by the tenant/iam contexts.
package lighting

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/masterfabric-go/masterfabric/internal/domain/lighting/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/lighting/repository"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

// SegmentRepo implements repository.SegmentRepository with PostgreSQL.
type SegmentRepo struct {
	db *pgxpool.Pool
}

// NewSegmentRepo creates a new SegmentRepo.
func NewSegmentRepo(db *pgxpool.Pool) *SegmentRepo {
	return &SegmentRepo{db: db}
}

const segmentColumns = `id, external_id, name, district, road_type, centroid_lat, centroid_lon,
	geometry, length_m, sample_count, street_light_count, pole_count, night_sample_ratio,
	tree_count, vegetation_ratio, building_ratio, road_width_m, sidewalk_ratio, sky_ratio, brightness_factor,
	lighting_density, recommended_density, adequacy,
	lighting_sufficiency, occlusion, infrastructure_adequacy, overall_score,
	risk_score, risk_level, created_at, updated_at`

func (r *SegmentRepo) Upsert(ctx context.Context, seg *model.StreetSegment) error {
	if seg.ID == uuid.Nil {
		seg.ID = uuid.New()
	}
	now := time.Now().UTC()
	if seg.CreatedAt.IsZero() {
		seg.CreatedAt = now
	}
	seg.UpdatedAt = now

	geometryJSON, err := json.Marshal(seg.Geometry)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to marshal geometry", err)
	}

	_, err = r.db.Exec(ctx,
		`INSERT INTO street_segments (`+segmentColumns+`)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,
		         $21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31)
		 ON CONFLICT (external_id) DO UPDATE SET
		   name=EXCLUDED.name, district=EXCLUDED.district, road_type=EXCLUDED.road_type,
		   centroid_lat=EXCLUDED.centroid_lat, centroid_lon=EXCLUDED.centroid_lon,
		   geometry=EXCLUDED.geometry, length_m=EXCLUDED.length_m, sample_count=EXCLUDED.sample_count,
		   street_light_count=EXCLUDED.street_light_count, pole_count=EXCLUDED.pole_count,
		   night_sample_ratio=EXCLUDED.night_sample_ratio,
		   tree_count=EXCLUDED.tree_count, vegetation_ratio=EXCLUDED.vegetation_ratio,
		   building_ratio=EXCLUDED.building_ratio, road_width_m=EXCLUDED.road_width_m,
		   sidewalk_ratio=EXCLUDED.sidewalk_ratio, sky_ratio=EXCLUDED.sky_ratio,
		   brightness_factor=EXCLUDED.brightness_factor,
		   lighting_density=EXCLUDED.lighting_density,
		   recommended_density=EXCLUDED.recommended_density, adequacy=EXCLUDED.adequacy,
		   lighting_sufficiency=EXCLUDED.lighting_sufficiency, occlusion=EXCLUDED.occlusion,
		   infrastructure_adequacy=EXCLUDED.infrastructure_adequacy, overall_score=EXCLUDED.overall_score,
		   risk_score=EXCLUDED.risk_score, risk_level=EXCLUDED.risk_level, updated_at=EXCLUDED.updated_at`,
		seg.ID, seg.ExternalID, seg.Name, seg.District, seg.RoadType, seg.CentroidLat, seg.CentroidLon,
		geometryJSON, seg.LengthM, seg.SampleCount, seg.StreetLightCount, seg.PoleCount, seg.NightSampleRatio,
		seg.TreeCount, seg.VegetationRatio, seg.BuildingRatio, seg.RoadWidthM, seg.SidewalkRatio, seg.SkyRatio, seg.BrightnessFactor,
		seg.LightingDensity, seg.RecommendedDensity, seg.Adequacy,
		seg.LightingSufficiency, seg.Occlusion, seg.InfrastructureAdequacy, seg.OverallScore,
		seg.RiskScore, seg.RiskLevel,
		seg.CreatedAt, seg.UpdatedAt,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to upsert street segment", err)
	}
	return nil
}

func scanSegment(row pgx.Row) (*model.StreetSegment, error) {
	var s model.StreetSegment
	var geometryJSON []byte
	err := row.Scan(
		&s.ID, &s.ExternalID, &s.Name, &s.District, &s.RoadType, &s.CentroidLat, &s.CentroidLon,
		&geometryJSON, &s.LengthM, &s.SampleCount, &s.StreetLightCount, &s.PoleCount, &s.NightSampleRatio,
		&s.TreeCount, &s.VegetationRatio, &s.BuildingRatio, &s.RoadWidthM, &s.SidewalkRatio, &s.SkyRatio, &s.BrightnessFactor,
		&s.LightingDensity, &s.RecommendedDensity, &s.Adequacy,
		&s.LightingSufficiency, &s.Occlusion, &s.InfrastructureAdequacy, &s.OverallScore,
		&s.RiskScore, &s.RiskLevel,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(geometryJSON) > 0 {
		_ = json.Unmarshal(geometryJSON, &s.Geometry)
	}
	return &s, nil
}

func (r *SegmentRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.StreetSegment, error) {
	row := r.db.QueryRow(ctx, `SELECT `+segmentColumns+` FROM street_segments WHERE id=$1`, id)
	seg, err := scanSegment(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErr.New(domainErr.ErrNotFound, "street segment not found", nil)
		}
		return nil, domainErr.New(domainErr.ErrInternal, "failed to get street segment", err)
	}
	return seg, nil
}

func (r *SegmentRepo) GetByExternalID(ctx context.Context, externalID string) (*model.StreetSegment, error) {
	row := r.db.QueryRow(ctx, `SELECT `+segmentColumns+` FROM street_segments WHERE external_id=$1`, externalID)
	seg, err := scanSegment(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErr.New(domainErr.ErrNotFound, "street segment not found", nil)
		}
		return nil, domainErr.New(domainErr.ErrInternal, "failed to get street segment", err)
	}
	return seg, nil
}

// buildWhere constructs a parameterized WHERE clause from a filter.
func buildWhere(filter repository.SegmentFilter) (string, []interface{}) {
	var clauses []string
	var args []interface{}
	i := 1
	add := func(clause string, val interface{}) {
		clauses = append(clauses, clause+"$"+itoa(i))
		args = append(args, val)
		i++
	}
	if filter.District != "" {
		add("district=", filter.District)
	}
	if filter.RoadType != "" {
		add("road_type=", filter.RoadType)
	}
	if filter.RiskLevel != "" {
		add("risk_level=", filter.RiskLevel)
	}
	if filter.MinRisk > 0 {
		add("risk_score>=", filter.MinRisk)
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func (r *SegmentRepo) List(ctx context.Context, filter repository.SegmentFilter, offset, limit int) ([]*model.StreetSegment, int, error) {
	where, args := buildWhere(filter)

	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM street_segments`+where, args...).Scan(&total); err != nil {
		return nil, 0, domainErr.New(domainErr.ErrInternal, "failed to count segments", err)
	}

	query := `SELECT ` + segmentColumns + ` FROM street_segments` + where +
		` ORDER BY risk_score DESC LIMIT $` + itoa(len(args)+1) + ` OFFSET $` + itoa(len(args)+2)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, domainErr.New(domainErr.ErrInternal, "failed to list segments", err)
	}
	defer rows.Close()

	segments, err := collectSegments(rows)
	if err != nil {
		return nil, 0, err
	}
	return segments, total, nil
}

func (r *SegmentRepo) ListAll(ctx context.Context, filter repository.SegmentFilter) ([]*model.StreetSegment, error) {
	where, args := buildWhere(filter)
	rows, err := r.db.Query(ctx, `SELECT `+segmentColumns+` FROM street_segments`+where+` ORDER BY risk_score DESC`, args...)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to list all segments", err)
	}
	defer rows.Close()
	return collectSegments(rows)
}

func collectSegments(rows pgx.Rows) ([]*model.StreetSegment, error) {
	var segments []*model.StreetSegment
	for rows.Next() {
		seg, err := scanSegment(rows)
		if err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "failed to scan segment", err)
		}
		segments = append(segments, seg)
	}
	return segments, nil
}

func (r *SegmentRepo) Count(ctx context.Context) (int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM street_segments`).Scan(&total); err != nil {
		return 0, domainErr.New(domainErr.ErrInternal, "failed to count segments", err)
	}
	return total, nil
}

func (r *SegmentRepo) Stats(ctx context.Context) (*model.CityStats, error) {
	stats := &model.CityStats{ByRiskLevel: map[string]int{}}

	var totalLengthM float64
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*), COALESCE(SUM(street_light_count),0), COALESCE(SUM(pole_count),0),
		        COALESCE(SUM(length_m),0), COALESCE(AVG(risk_score),0)
		 FROM street_segments`,
	).Scan(&stats.TotalSegments, &stats.TotalStreetLights, &stats.TotalPoles, &totalLengthM, &stats.AvgRiskScore)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to compute stats", err)
	}
	stats.TotalLengthKm = round2(totalLengthM / 1000.0)
	stats.AvgRiskScore = round1(stats.AvgRiskScore)

	// By risk level.
	levelRows, err := r.db.Query(ctx, `SELECT risk_level, COUNT(*) FROM street_segments GROUP BY risk_level`)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to group by risk level", err)
	}
	defer levelRows.Close()
	for levelRows.Next() {
		var level string
		var count int
		if err := levelRows.Scan(&level, &count); err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "failed to scan risk level", err)
		}
		stats.ByRiskLevel[level] = count
		if level == string(model.RiskHigh) || level == string(model.RiskCritical) {
			stats.HighRiskSegments += count
		}
	}

	// By district.
	distRows, err := r.db.Query(ctx,
		`SELECT district, COUNT(*), COALESCE(AVG(risk_score),0), COALESCE(SUM(street_light_count),0),
		        COALESCE(SUM(CASE WHEN risk_level IN ('high','critical') THEN 1 ELSE 0 END),0)
		 FROM street_segments GROUP BY district ORDER BY AVG(risk_score) DESC`)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to group by district", err)
	}
	defer distRows.Close()
	for distRows.Next() {
		var d model.DistrictStat
		if err := distRows.Scan(&d.District, &d.SegmentCount, &d.AvgRiskScore, &d.TotalStreetLights, &d.HighRiskSegments); err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "failed to scan district stat", err)
		}
		d.AvgRiskScore = round1(d.AvgRiskScore)
		stats.ByDistrict = append(stats.ByDistrict, d)
	}

	return stats, nil
}
