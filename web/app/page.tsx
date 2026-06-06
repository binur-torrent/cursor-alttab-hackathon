"use client";

import { useEffect, useState } from "react";
import MapView from "@/components/MapView";
import StatsBar from "@/components/StatsBar";
import SegmentPanel from "@/components/SegmentPanel";
import { api } from "@/lib/api";
import type { CityStats, SegmentDetail, StreetSegment } from "@/lib/types";

const RISK_FILTERS = [
  { value: "", label: "All segments" },
  { value: "critical", label: "Critical" },
  { value: "high", label: "High" },
  { value: "medium", label: "Medium" },
  { value: "low", label: "Low" },
];

export default function Dashboard() {
  const [stats, setStats] = useState<CityStats | null>(null);
  const [segments, setSegments] = useState<StreetSegment[]>([]);
  const [selected, setSelected] = useState<StreetSegment | null>(null);
  const [detail, setDetail] = useState<SegmentDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [riskFilter, setRiskFilter] = useState("");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api.stats().then(setStats).catch((e) => setError(String(e)));
  }, []);

  useEffect(() => {
    api
      .mapSegments(riskFilter || undefined)
      .then((r) => setSegments(r.segments))
      .catch((e) => setError(String(e)));
  }, [riskFilter]);

  async function handleSelect(seg: StreetSegment) {
    setSelected(seg);
    setDetailLoading(true);
    try {
      const d = await api.segment(seg.external_id);
      setDetail(d);
    } catch (e) {
      setError(String(e));
    } finally {
      setDetailLoading(false);
    }
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-[1400px] flex-col gap-4 p-4 lg:p-6">
      {/* Header */}
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="flex items-center gap-2 text-xl font-bold tracking-tight">
            <span className="inline-block h-3 w-3 rounded-full bg-amber-400 shadow-[0_0_12px_2px_rgba(251,191,36,0.7)]" />
            LumiCity AI
          </h1>
          <p className="text-sm text-slate-400">
            Istanbul streetlight intelligence · adaptive-lighting decision support
          </p>
        </div>
        <select
          value={riskFilter}
          onChange={(e) => setRiskFilter(e.target.value)}
          className="rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-200 outline-none focus:border-amber-400"
        >
          {RISK_FILTERS.map((f) => (
            <option key={f.value} value={f.value}>
              {f.label}
            </option>
          ))}
        </select>
      </header>

      {error && (
        <div className="rounded-lg border border-red-900 bg-red-950/50 px-4 py-2 text-sm text-red-300">
          Could not reach the backend ({error}). Is the API running at the configured URL?
        </div>
      )}

      <StatsBar stats={stats} />

      {/* Map + detail */}
      <div className="grid flex-1 grid-cols-1 gap-4 lg:grid-cols-[1fr_380px]">
        <div className="h-[520px] overflow-hidden rounded-2xl border border-slate-800 lg:h-auto lg:min-h-[560px]">
          <MapView
            segments={segments}
            selectedId={selected?.external_id}
            onSelect={handleSelect}
          />
        </div>
        <aside className="rounded-2xl border border-slate-800 bg-slate-900/30 p-4">
          <SegmentPanel detail={detail} loading={detailLoading} />
        </aside>
      </div>

      <footer className="pb-2 text-center text-xs text-slate-600">
        Data: HuggingFace street imagery · detection: Mask2Former (Mapillary-Vistas) ·
        backend: masterfabric-go · KVKK compliant
      </footer>
    </main>
  );
}
