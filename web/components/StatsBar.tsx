"use client";

import type { CityStats, RiskLevel } from "@/lib/types";
import { RISK_COLORS, num } from "@/lib/format";

function Kpi({ label, value, sub }: { label: string; value: string; sub?: string }) {
  return (
    <div className="rounded-xl border border-slate-800 bg-slate-900/60 px-4 py-3">
      <div className="text-2xl font-semibold text-slate-50">{value}</div>
      <div className="text-xs uppercase tracking-wide text-slate-400">{label}</div>
      {sub && <div className="mt-0.5 text-xs text-slate-500">{sub}</div>}
    </div>
  );
}

export default function StatsBar({ stats }: { stats: CityStats | null }) {
  if (!stats) {
    return (
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
        {Array.from({ length: 5 }).map((_, i) => (
          <div key={i} className="h-[72px] animate-pulse rounded-xl bg-slate-900/60" />
        ))}
      </div>
    );
  }

  const total = stats.total_segments || 1;
  const order: RiskLevel[] = ["critical", "high", "medium", "low"];

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
        <Kpi label="Segments" value={num(stats.total_segments)} sub={`${num(stats.total_length_km, 1)} km mapped`} />
        <Kpi label="Streetlights" value={num(stats.total_street_lights)} sub={`${num(stats.total_poles)} poles`} />
        <Kpi
          label="Avg lighting score"
          value={num(100 - stats.avg_risk_score, 0)}
          sub="0 = dark · 100 = well lit"
        />
        <Kpi
          label="High-risk"
          value={num(stats.high_risk_segments)}
          sub={`${num((stats.high_risk_segments / total) * 100, 0)}% of network`}
        />
        <Kpi label="Districts" value={num(stats.by_district.length)} sub="covered" />
      </div>

      {/* Risk distribution bar */}
      <div className="flex h-3 w-full overflow-hidden rounded-full bg-slate-800">
        {order.map((lvl) => {
          const count = stats.by_risk_level[lvl] ?? 0;
          const width = (count / total) * 100;
          if (width === 0) return null;
          return (
            <div
              key={lvl}
              style={{ width: `${width}%`, backgroundColor: RISK_COLORS[lvl] }}
              title={`${lvl}: ${count}`}
            />
          );
        })}
      </div>
      <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-400">
        {order.map((lvl) => (
          <span key={lvl} className="flex items-center gap-1.5">
            <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: RISK_COLORS[lvl] }} />
            {lvl} · {num(stats.by_risk_level[lvl] ?? 0)}
          </span>
        ))}
      </div>
    </div>
  );
}
