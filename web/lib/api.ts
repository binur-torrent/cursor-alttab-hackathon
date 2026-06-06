import type {
  AnalyzeResult,
  AnalyzeSegmentResult,
  CityStats,
  MapResponse,
  Paginated,
  RescoreRequest,
  RescoreResult,
  ScenarioParams,
  ScenarioResult,
  SegmentDetail,
  StreetSegment,
} from "./types";

// Backend base URL. Defaults to the local masterfabric-go server; override with
// NEXT_PUBLIC_API_URL (e.g. the Render deployment) in production.
export const API_BASE =
  process.env.NEXT_PUBLIC_API_URL?.replace(/\/$/, "") || "http://localhost:8081";

async function getJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, { cache: "no-store" });
  if (!res.ok) throw new Error(`GET ${path} failed: ${res.status}`);
  return res.json() as Promise<T>;
}

async function postJSON<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error(`POST ${path} failed: ${res.status}`);
  return res.json() as Promise<T>;
}

export const api = {
  stats: () => getJSON<CityStats>("/api/v1/lighting/stats"),
  mapSegments: (riskLevel?: string) =>
    getJSON<MapResponse>(
      `/api/v1/lighting/segments/map${riskLevel ? `?risk_level=${riskLevel}` : ""}`,
    ),
  listSegments: (params: Record<string, string | number> = {}) => {
    const qs = new URLSearchParams(
      Object.entries(params).map(([k, v]) => [k, String(v)]),
    ).toString();
    return getJSON<Paginated<StreetSegment>>(
      `/api/v1/lighting/segments${qs ? `?${qs}` : ""}`,
    );
  },
  segment: (id: string) => getJSON<SegmentDetail>(`/api/v1/lighting/segments/${id}`),
  rescore: (id: string, body: RescoreRequest) =>
    postJSON<RescoreResult>(`/api/v1/lighting/segments/${id}/rescore`, body),
  simulate: (params: ScenarioParams) =>
    postJSON<ScenarioResult>("/api/v1/lighting/simulate", params),
  analyze: (body: {
    lat: number;
    lon: number;
    heading?: number;
    road_type?: string;
    length_m?: number;
    is_night?: boolean;
    address?: string;
  }) => postJSON<AnalyzeResult>("/api/v1/lighting/analyze", body),
  analyzeSegment: (body: {
    lat: number;
    lon: number;
    road_type?: string;
    length_m?: number;
    is_night?: boolean;
    address?: string;
  }) => postJSON<AnalyzeSegmentResult>("/api/v1/lighting/segments/analyze", body),
};
