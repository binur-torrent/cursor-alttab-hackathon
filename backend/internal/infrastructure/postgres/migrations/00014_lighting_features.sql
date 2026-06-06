-- +goose Up
-- Scoring v2: environmental features detected from imagery plus the three
-- headline scores and the composite overall score. Mirrors EnsureSchema in
-- internal/infrastructure/postgres/lighting/schema.go.
ALTER TABLE street_segments ADD COLUMN IF NOT EXISTS tree_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE street_segments ADD COLUMN IF NOT EXISTS vegetation_ratio DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE street_segments ADD COLUMN IF NOT EXISTS building_ratio DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE street_segments ADD COLUMN IF NOT EXISTS road_width_m DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE street_segments ADD COLUMN IF NOT EXISTS sidewalk_ratio DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE street_segments ADD COLUMN IF NOT EXISTS sky_ratio DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE street_segments ADD COLUMN IF NOT EXISTS brightness_factor DOUBLE PRECISION NOT NULL DEFAULT 1;
ALTER TABLE street_segments ADD COLUMN IF NOT EXISTS lighting_sufficiency DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE street_segments ADD COLUMN IF NOT EXISTS occlusion DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE street_segments ADD COLUMN IF NOT EXISTS infrastructure_adequacy DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE street_segments ADD COLUMN IF NOT EXISTS overall_score DOUBLE PRECISION NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE street_segments DROP COLUMN IF EXISTS tree_count;
ALTER TABLE street_segments DROP COLUMN IF EXISTS vegetation_ratio;
ALTER TABLE street_segments DROP COLUMN IF EXISTS building_ratio;
ALTER TABLE street_segments DROP COLUMN IF EXISTS road_width_m;
ALTER TABLE street_segments DROP COLUMN IF EXISTS sidewalk_ratio;
ALTER TABLE street_segments DROP COLUMN IF EXISTS sky_ratio;
ALTER TABLE street_segments DROP COLUMN IF EXISTS brightness_factor;
ALTER TABLE street_segments DROP COLUMN IF EXISTS lighting_sufficiency;
ALTER TABLE street_segments DROP COLUMN IF EXISTS occlusion;
ALTER TABLE street_segments DROP COLUMN IF EXISTS infrastructure_adequacy;
ALTER TABLE street_segments DROP COLUMN IF EXISTS overall_score;
