export type RiskLevel = "low" | "medium" | "high" | "critical";

export interface StreetSegment {
  id: string;
  external_id: string;
  name: string;
  district: string;
  road_type: string;
  centroid_lat: number;
  centroid_lon: number;
  length_m: number;
  street_light_count: number;
  pole_count: number;
  adequacy: number;
  risk_score: number;
  risk_level: RiskLevel;
}

export interface LightingAnalysis {
  id: string;
  time_of_day: string;
  street_light_count: number;
  backend: string;
  anonymized: boolean;
  faces_blurred: number;
  plates_blurred: number;
}

export interface SegmentDetail {
  segment: StreetSegment;
  fixtures: { id: string }[];
  analyses: LightingAnalysis[];
}

export interface Paginated<T> {
  data: T[];
  total_count: number;
}

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
  source: string;
}
