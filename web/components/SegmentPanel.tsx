"use client";

import type { SegmentDetail } from "@/lib/types";
import { num } from "@/lib/format";
import RiskBadge from "./RiskBadge";

function Metric({ label, value, hint }: { label: string; value: string; hint?: string }) {
  return (
    <div className="rounded-lg border border-slate-800 bg-slate-900/40 px-3 py-2">
      <div className="text-xs text-slate-400">{label}</div>
      <div className="text-sm font-medium text-slate-100">{value}</div>
      {hint && <div className="text-[11px] text-slate-500">{hint}</div>}
    </div>
  );
}

export default function SegmentPanel({
  detail,
  loading,
}: {
  detail: SegmentDetail | null;
  loading: boolean;
}) {
  if (loading) {
    return <div className="h-40 animate-pulse rounded-xl bg-slate-900/60" />;
  }
  if (!detail) {
    return (
      <div className="rounded-xl border border-dashed border-slate-800 bg-slate-900/30 p-6 text-center text-sm text-slate-500">
        Select a street segment on the map to inspect its lighting analysis.
      </div>
    );
  }

  const s = detail.segment;
  const adequacyPct = Math.round(s.adequacy * 100);
  const nightPct = Math.round(s.night_sample_ratio * 100);
  const totalBlurred = detail.analyses.reduce(
    (acc, a) => acc + a.faces_blurred + a.plates_blurred,
    0,
  );

  return (
    <div className="space-y-4">
      <div>
        <div className="flex items-start justify-between gap-2">
          <h2 className="text-lg font-semibold text-slate-50">{s.name}</h2>
          <RiskBadge level={s.risk_level} score={s.risk_score} />
        </div>
        <p className="text-sm text-slate-400">
          {s.district} · {s.road_type} · {num(s.length_m)} m
        </p>
      </div>

      <div className="grid grid-cols-2 gap-2">
        <Metric label="Streetlights" value={num(s.street_light_count)} hint={`${num(s.pole_count)} poles`} />
        <Metric
          label="Lighting density"
          value={`${num(s.lighting_density, 2)}/100m`}
          hint={`target ${num(s.recommended_density, 1)}`}
        />
        <Metric label="Adequacy" value={`${adequacyPct}%`} hint="of recommended" />
        <Metric label="Night samples" value={`${nightPct}%`} hint={`${s.sample_count} frames`} />
      </div>

      {/* Adequacy bar */}
      <div>
        <div className="mb-1 flex justify-between text-xs text-slate-400">
          <span>Illumination adequacy</span>
          <span>{adequacyPct}%</span>
        </div>
        <div className="h-2 w-full overflow-hidden rounded-full bg-slate-800">
          <div
            className="h-full rounded-full"
            style={{
              width: `${adequacyPct}%`,
              backgroundColor: adequacyPct >= 75 ? "#22c55e" : adequacyPct >= 40 ? "#eab308" : "#ef4444",
            }}
          />
        </div>
      </div>

      {/* Anonymized frames */}
      <div>
        <div className="mb-2 flex items-center justify-between">
          <h3 className="text-sm font-medium text-slate-200">
            Analyzed frames ({detail.analyses.length})
          </h3>
          <span className="text-xs text-emerald-400">
            {totalBlurred} faces/plates blurred
          </span>
        </div>
        <div className="max-h-44 space-y-1.5 overflow-auto pr-1">
          {detail.analyses.map((a) => (
            <div
              key={a.id}
              className="flex items-center justify-between rounded-lg border border-slate-800 bg-slate-900/40 px-3 py-1.5 text-xs"
            >
              <span className="text-slate-300">
                {a.time_of_day || "frame"} · {a.street_light_count} lights
              </span>
              <span className="flex items-center gap-2 text-slate-500">
                <span title="anonymized" className="text-emerald-400">
                  ●
                </span>
                {a.backend}
              </span>
            </div>
          ))}
        </div>
      </div>

      <p className="rounded-lg bg-slate-900/60 px-3 py-2 text-[11px] leading-relaxed text-slate-500">
        KVKK: only urban assets are detected. Faces and license plates in source
        frames are irreversibly blurred before storage. No raw imagery is retained.
      </p>
    </div>
  );
}
