-- +goose Up
CREATE TABLE IF NOT EXISTS street_segments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id         VARCHAR(100) NOT NULL UNIQUE,
    name                VARCHAR(255) NOT NULL DEFAULT '',
    district            VARCHAR(120) NOT NULL DEFAULT '',
    road_type           VARCHAR(40) NOT NULL DEFAULT 'secondary',
    centroid_lat        DOUBLE PRECISION NOT NULL,
    centroid_lon        DOUBLE PRECISION NOT NULL,
    geometry            JSONB,
    length_m            DOUBLE PRECISION NOT NULL DEFAULT 0,
    sample_count        INTEGER NOT NULL DEFAULT 0,
    street_light_count  INTEGER NOT NULL DEFAULT 0,
    pole_count          INTEGER NOT NULL DEFAULT 0,
    night_sample_ratio  DOUBLE PRECISION NOT NULL DEFAULT 0,
    lighting_density    DOUBLE PRECISION NOT NULL DEFAULT 0,
    recommended_density DOUBLE PRECISION NOT NULL DEFAULT 0,
    adequacy            DOUBLE PRECISION NOT NULL DEFAULT 0,
    risk_score          DOUBLE PRECISION NOT NULL DEFAULT 0,
    risk_level          VARCHAR(20) NOT NULL DEFAULT 'low',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_street_segments_risk_level ON street_segments(risk_level);
CREATE INDEX IF NOT EXISTS idx_street_segments_district ON street_segments(district);
CREATE INDEX IF NOT EXISTS idx_street_segments_risk_score ON street_segments(risk_score DESC);

CREATE TABLE IF NOT EXISTS light_fixtures (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    segment_id  UUID NOT NULL REFERENCES street_segments(id) ON DELETE CASCADE,
    type        VARCHAR(40) NOT NULL DEFAULT 'street_light',
    lat         DOUBLE PRECISION NOT NULL,
    lon         DOUBLE PRECISION NOT NULL,
    confidence  DOUBLE PRECISION NOT NULL DEFAULT 0,
    source      VARCHAR(60) NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_light_fixtures_segment ON light_fixtures(segment_id);

CREATE TABLE IF NOT EXISTS lighting_analyses (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    segment_id         UUID NOT NULL REFERENCES street_segments(id) ON DELETE CASCADE,
    external_id        VARCHAR(120) NOT NULL DEFAULT '',
    lat                DOUBLE PRECISION NOT NULL,
    lon                DOUBLE PRECISION NOT NULL,
    heading            DOUBLE PRECISION NOT NULL DEFAULT 0,
    captured_at        TIMESTAMPTZ,
    time_of_day        VARCHAR(20) NOT NULL DEFAULT '',
    road_type          VARCHAR(40) NOT NULL DEFAULT '',
    street_light_count INTEGER NOT NULL DEFAULT 0,
    pole_count         INTEGER NOT NULL DEFAULT 0,
    anonymized         BOOLEAN NOT NULL DEFAULT TRUE,
    faces_blurred      INTEGER NOT NULL DEFAULT 0,
    plates_blurred     INTEGER NOT NULL DEFAULT 0,
    backend            VARCHAR(60) NOT NULL DEFAULT '',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_lighting_analyses_segment ON lighting_analyses(segment_id);

-- +goose Down
DROP TABLE IF EXISTS lighting_analyses;
DROP TABLE IF EXISTS light_fixtures;
DROP TABLE IF EXISTS street_segments;
