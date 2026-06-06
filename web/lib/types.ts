// Types mirroring the Go backend JSON (internal/domain/lighting/model).

export type RiskLevel = "low" | "medium" | "high" | "critical";

export interface StreetSegment {
  id: string;
  external_id: string;
  name: string;
  district: string;
  road_type: string;
  centroid_lat: number;
  centroid_lon: number;
  geometry: [number, number][]; // [[lat, lon], ...]
  length_m: number;
  sample_count: number;
  street_light_count: number;
  pole_count: number;
  night_sample_ratio: number;
  lighting_density: number;
  recommended_density: number;
  adequacy: number;
  risk_score: number;
  risk_level: RiskLevel;
  created_at?: string;
  updated_at?: string;
}

export interface LightFixture {
  id: string;
  segment_id: string;
  type: string;
  lat: number;
  lon: number;
  confidence: number;
  source: string;
}

export interface LightingAnalysis {
  id: string;
  segment_id: string;
  external_id: string;
  lat: number;
  lon: number;
  heading: number;
  captured_at?: string;
  time_of_day: string;
  road_type: string;
  street_light_count: number;
  pole_count: number;
  anonymized: boolean;
  faces_blurred: number;
  plates_blurred: number;
  backend: string;
}

export interface SegmentDetail {
  segment: StreetSegment;
  fixtures: LightFixture[];
  analyses: LightingAnalysis[];
}

export interface DistrictStat {
  district: string;
  segment_count: number;
  avg_risk_score: number;
  total_street_lights: number;
  high_risk_segments: number;
}

export interface CityStats {
  total_segments: number;
  total_street_lights: number;
  total_poles: number;
  total_length_km: number;
  avg_risk_score: number;
  by_risk_level: Record<string, number>;
  by_district: DistrictStat[];
  high_risk_segments: number;
}

export interface Paginated<T> {
  data: T[];
  page: number;
  per_page: number;
  total_count: number;
  total_pages: number;
}

export interface MapResponse {
  count: number;
  segments: StreetSegment[];
}

// Simulation (POST /lighting/simulate)
export interface ScenarioParams {
  target_risk_levels: string[];
  target_adequacy: number;
  night_dimming_percent: number;
  fixture_wattage?: number;
  hours_per_night?: number;
  energy_cost_per_kwh?: number;
  co2_kg_per_kwh?: number;
}

export interface ScenarioResult {
  segments_total: number;
  segments_upgraded: number;
  segments_dimmed: number;
  baseline_fixtures: number;
  proposed_fixtures: number;
  added_fixtures: number;
  baseline_avg_risk: number;
  proposed_avg_risk: number;
  risk_reduction_percent: number;
  baseline_energy_kwh_year: number;
  proposed_energy_kwh_year: number;
  energy_delta_kwh_year: number;
  energy_saved_percent: number;
  annual_cost_delta: number;
  co2_delta_kg_year: number;
}

// Live analysis (POST /lighting/analyze)
export interface AnalyzeResult {
  street_light_count: number;
  pole_count: number;
  detector_backend: string;
  faces_blurred: number;
  plates_blurred: number;
  anonymized: boolean;
  risk_score: number;
  risk_level: RiskLevel;
  adequacy: number;
  lighting_density: number;
  image_base64?: string | null;
  lat?: number;
  lon?: number;
  address?: string;
}
