"use client";

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";
import type { ScenarioResult } from "@/lib/types";
import { num, pct } from "@/lib/format";

const ALL_LEVELS = ["critical", "high", "medium"] as const;

function Delta({
  label,
  value,
  goodWhenNegative = false,
  suffix = "",
}: {
  label: string;
  value: number;
  goodWhenNegative?: boolean;
  suffix?: string;
}) {
  const good = goodWhenNegative ? value < 0 : value > 0;
  const color = value === 0 ? "#94a3b8" : good ? "#22c55e" : "#f97316";
  return (
    <div className="rounded-lg border border-slate-800 bg-slate-900/40 px-3 py-2">
      <div className="text-xs text-slate-400">{label}</div>
      <div className="text-lg font-semibold" style={{ color }}>
        {value > 0 ? "+" : ""}
        {num(value, 1)}
        {suffix}
      </div>
    </div>
  );
}

export default function SimulationPanel({ districts }: { districts: string[] }) {
  const [targetAdequacy, setTargetAdequacy] = useState(0.8);
  const [nightDimming, setNightDimming] = useState(50);
  const [levels, setLevels] = useState<Record<string, boolean>>({
    critical: true,
    high: true,
    medium: false,
  });
  const [district, setDistrict] = useState("");
  const [result, setResult] = useState<ScenarioResult | null>(null);
  const [loading, setLoading] = useState(false);

  const run = useCallback(async () => {
    setLoading(true);
    try {
      const res = await api.simulate({
        target_risk_levels: ALL_LEVELS.filter((l) => levels[l]),
        target_adequacy: targetAdequacy,
        night_dimming_percent: nightDimming,
        district: district || undefined,
      } as never);
      setResult(res);
    } finally {
      setLoading(false);
    }
  }, [targetAdequacy, nightDimming, levels, district]);

  useEffect(() => {
    const t = setTimeout(run, 250);
    return () => clearTimeout(t);
  }, [run]);

  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-lg font-semibold text-slate-50">Adaptive lighting simulation</h2>
        <p className="text-sm text-slate-400">
          Upgrade underlit high-risk streets and dim well-lit ones at night.
        </p>
      </div>

      {/* District scope */}
      <label className="block">
        <span className="text-xs text-slate-400">Scope</span>
        <select
          value={district}
          onChange={(e) => setDistrict(e.target.value)}
          className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-900 px-2 py-1.5 text-sm text-slate-200 outline-none focus:border-amber-400"
        >
          <option value="">Whole city</option>
          {districts.map((d) => (
            <option key={d} value={d}>
              {d}
            </option>
          ))}
        </select>
      </label>

      {/* Target risk levels */}
      <div>
        <span className="text-xs text-slate-400">Upgrade segments at risk</span>
        <div className="mt-1 flex gap-2">
          {ALL_LEVELS.map((l) => (
            <button
              key={l}
              onClick={() => setLevels((s) => ({ ...s, [l]: !s[l] }))}
              className={`rounded-full border px-3 py-1 text-xs capitalize transition ${
                levels[l]
                  ? "border-amber-400 bg-amber-400/15 text-amber-300"
                  : "border-slate-700 text-slate-400"
              }`}
            >
              {l}
            </button>
          ))}
        </div>
      </div>

      {/* Target adequacy slider */}
      <label className="block">
        <div className="flex justify-between text-xs text-slate-400">
          <span>Target illumination adequacy</span>
          <span className="text-slate-200">{Math.round(targetAdequacy * 100)}%</span>
        </div>
        <input
          type="range"
          min={0.5}
          max={1}
          step={0.05}
          value={targetAdequacy}
          onChange={(e) => setTargetAdequacy(Number(e.target.value))}
          className="mt-1 w-full accent-amber-400"
        />
      </label>

      {/* Night dimming slider */}
      <label className="block">
        <div className="flex justify-between text-xs text-slate-400">
          <span>Late-night dimming (well-lit streets)</span>
          <span className="text-slate-200">{nightDimming}%</span>
        </div>
        <input
          type="range"
          min={0}
          max={80}
          step={5}
          value={nightDimming}
          onChange={(e) => setNightDimming(Number(e.target.value))}
          className="mt-1 w-full accent-sky-400"
        />
      </label>

      {/* Results */}
      {result && (
        <div className={`space-y-3 transition ${loading ? "opacity-50" : ""}`}>
          <div className="grid grid-cols-2 gap-2">
            <Delta label="Risk reduction" value={result.risk_reduction_percent} suffix="%" />
            <Delta
              label="Energy"
              value={result.energy_delta_kwh_year > 0 ? -result.energy_saved_percent : result.energy_saved_percent}
              suffix="%"
              goodWhenNegative={false}
            />
          </div>
          <div className="grid grid-cols-2 gap-2 text-sm">
            <div className="rounded-lg border border-slate-800 bg-slate-900/40 px-3 py-2">
              <div className="text-xs text-slate-400">Fixtures added</div>
              <div className="font-medium text-slate-100">+{num(result.added_fixtures)}</div>
              <div className="text-[11px] text-slate-500">
                {num(result.segments_upgraded)} segments upgraded
              </div>
            </div>
            <div className="rounded-lg border border-slate-800 bg-slate-900/40 px-3 py-2">
              <div className="text-xs text-slate-400">Avg risk</div>
              <div className="font-medium text-slate-100">
                {num(result.baseline_avg_risk, 1)} → {num(result.proposed_avg_risk, 1)}
              </div>
              <div className="text-[11px] text-slate-500">{num(result.segments_dimmed)} dimmed</div>
            </div>
          </div>
          <div className="rounded-lg border border-slate-800 bg-slate-900/40 px-3 py-2 text-sm">
            <div className="flex justify-between">
              <span className="text-slate-400">Annual energy</span>
              <span className="text-slate-200">
                {num(result.proposed_energy_kwh_year / 1000, 1)} MWh/yr
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-400">Annual cost change</span>
              <span className={result.annual_cost_delta <= 0 ? "text-emerald-400" : "text-amber-400"}>
                {result.annual_cost_delta <= 0 ? "" : "+"}
                {num(result.annual_cost_delta)} TRY
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-400">CO₂ change</span>
              <span className={result.co2_delta_kg_year <= 0 ? "text-emerald-400" : "text-amber-400"}>
                {result.co2_delta_kg_year <= 0 ? "" : "+"}
                {num(result.co2_delta_kg_year)} kg/yr
              </span>
            </div>
          </div>
          <p className="text-[11px] leading-relaxed text-slate-500">
            Baseline vs. proposed across {num(result.segments_total)} segments. Adding
            light raises energy use but cuts safety risk; night dimming on already
            well-lit streets recovers energy. Tune the levers to balance both.
          </p>
        </div>
      )}
    </div>
  );
}
