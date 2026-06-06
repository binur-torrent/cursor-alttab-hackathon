import type { AnalyzeResult, Paginated, SegmentDetail, StreetSegment } from "./types";

// Backend base URL. Set EXPO_PUBLIC_API_URL (e.g. your Render deployment, or your
// machine's LAN IP for local dev — "localhost" does not resolve on a device).
export const API_BASE =
  process.env.EXPO_PUBLIC_API_URL?.replace(/\/$/, "") || "http://localhost:8081";

async function getJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`);
  if (!res.ok) throw new Error(`GET ${path} -> ${res.status}`);
  return res.json() as Promise<T>;
}

export const api = {
  problemSegments: () =>
    getJSON<Paginated<StreetSegment>>(
      "/api/v1/lighting/segments?min_risk=50&per_page=100",
    ),
  allSegments: () =>
    getJSON<{ count: number; segments: StreetSegment[] }>(
      "/api/v1/lighting/segments/map",
    ),
  segment: (id: string) => getJSON<SegmentDetail>(`/api/v1/lighting/segments/${id}`),
  analyze: (body: {
    lat: number;
    lon: number;
    road_type?: string;
    is_night?: boolean;
    address?: string;
  }) =>
    fetch(`${API_BASE}/api/v1/lighting/analyze`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }).then((r) => {
      if (!r.ok) throw new Error(`analyze -> ${r.status}`);
      return r.json() as Promise<AnalyzeResult>;
    }),
};
