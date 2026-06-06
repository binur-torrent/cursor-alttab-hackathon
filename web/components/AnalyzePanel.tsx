"use client";

import { useState } from "react";
import { api } from "@/lib/api";
import type { AnalyzeResult } from "@/lib/types";
import { num } from "@/lib/format";
import RiskBadge from "./RiskBadge";

const LATLON_RE = /^\s*(-?\d+(?:\.\d+)?)\s*,\s*(-?\d+(?:\.\d+)?)\s*$/;

async function geocode(query: string): Promise<{ lat: number; lon: number } | null> {
  const m = query.match(LATLON_RE);
  if (m) return { lat: parseFloat(m[1]), lon: parseFloat(m[2]) };

  const url = `https://nominatim.openstreetmap.org/search?format=json&limit=1&countrycodes=tr&q=${encodeURIComponent(
    query + ", İstanbul",
  )}`;
  const res = await fetch(url, { headers: { Accept: "application/json" } });
  if (!res.ok) return null;
  const data = (await res.json()) as { lat: string; lon: string }[];
  if (!data.length) return null;
  return { lat: parseFloat(data[0].lat), lon: parseFloat(data[0].lon) };
}

export default function AnalyzePanel({
  onLocate,
}: {
  onLocate: (lat: number, lon: number) => void;
}) {
  const [query, setQuery] = useState("");
  const [isNight, setIsNight] = useState(true);
  const [roadType, setRoadType] = useState("secondary");
  const [result, setResult] = useState<AnalyzeResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function analyze() {
    if (!query.trim()) return;
    setLoading(true);
    setError(null);
    setResult(null);
    try {
      const loc = await geocode(query);
      if (!loc) {
        setError("Could not find that address in İstanbul.");
        return;
      }
      onLocate(loc.lat, loc.lon);
      const res = await api.analyze({
        lat: loc.lat,
        lon: loc.lon,
        road_type: roadType,
        is_night: isNight,
        address: query,
      });
      setResult(res);
    } catch (e) {
      setError(String(e));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-lg font-semibold text-slate-50">Analyze any address</h2>
        <p className="text-sm text-slate-400">
          On-demand lighting analysis from street imagery. Type an address or
          &ldquo;lat, lon&rdquo;.
        </p>
      </div>

      <div className="space-y-2">
        <input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && analyze()}
          placeholder="e.g. Bağdat Caddesi, Kadıköy"
          className="w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 outline-none focus:border-amber-400"
        />
        <div className="flex items-center gap-2">
          <select
            value={roadType}
            onChange={(e) => setRoadType(e.target.value)}
            className="rounded-lg border border-slate-700 bg-slate-900 px-2 py-1.5 text-xs text-slate-200 outline-none focus:border-amber-400"
          >
            {["motorway", "primary", "secondary", "residential", "pedestrian"].map((rt) => (
              <option key={rt} value={rt}>
                {rt}
              </option>
            ))}
          </select>
          <button
            onClick={() => setIsNight((v) => !v)}
            className={`rounded-full border px-3 py-1.5 text-xs transition ${
              isNight
                ? "border-sky-400 bg-sky-400/15 text-sky-300"
                : "border-slate-700 text-slate-400"
            }`}
          >
            {isNight ? "Night" : "Day"}
          </button>
          <button
            onClick={analyze}
            disabled={loading}
            className="ml-auto rounded-lg bg-amber-400 px-4 py-1.5 text-sm font-medium text-slate-950 transition hover:bg-amber-300 disabled:opacity-50"
          >
            {loading ? "Analyzing…" : "Analyze"}
          </button>
        </div>
      </div>

      {error && (
        <div className="rounded-lg border border-red-900 bg-red-950/50 px-3 py-2 text-sm text-red-300">
          {error}
        </div>
      )}

      {result && (
        <div className="space-y-3">
          {result.image_base64 ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={`data:image/jpeg;base64,${result.image_base64}`}
              alt="Anonymized street view"
              className="w-full rounded-lg border border-slate-800"
            />
          ) : (
            <div className="flex h-28 items-center justify-center rounded-lg border border-dashed border-slate-800 text-xs text-slate-500">
              {result.detector_backend === "heuristic-fallback"
                ? "No AI worker configured — showing modeled estimate"
                : "No image returned"}
            </div>
          )}

          <div className="flex items-center justify-between">
            <RiskBadge level={result.risk_level} score={result.risk_score} />
            <span className="text-xs text-slate-500">via {result.source}</span>
          </div>

          <div className="grid grid-cols-2 gap-2 text-sm">
            <div className="rounded-lg border border-slate-800 bg-slate-900/40 px-3 py-2">
              <div className="text-xs text-slate-400">Streetlights</div>
              <div className="font-medium text-slate-100">{num(result.street_light_count)}</div>
            </div>
            <div className="rounded-lg border border-slate-800 bg-slate-900/40 px-3 py-2">
              <div className="text-xs text-slate-400">Poles</div>
              <div className="font-medium text-slate-100">{num(result.pole_count)}</div>
            </div>
            <div className="rounded-lg border border-slate-800 bg-slate-900/40 px-3 py-2">
              <div className="text-xs text-slate-400">Adequacy</div>
              <div className="font-medium text-slate-100">{Math.round(result.adequacy * 100)}%</div>
            </div>
            <div className="rounded-lg border border-slate-800 bg-slate-900/40 px-3 py-2">
              <div className="text-xs text-slate-400">Detector</div>
              <div className="truncate text-xs font-medium text-slate-100">
                {result.detector_backend}
              </div>
            </div>
          </div>

          <p className="rounded-lg bg-slate-900/60 px-3 py-2 text-[11px] leading-relaxed text-slate-500">
            KVKK: {result.faces_blurred + result.plates_blurred} faces/plates
            irreversibly blurred · only urban assets detected · raw imagery not stored.
          </p>
        </div>
      )}
    </div>
  );
}
