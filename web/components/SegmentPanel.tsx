"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "@/lib/api";
import type {
  Recommendation,
  RescoreResult,
  SegmentDetail,
  StreetSegment,
} from "@/lib/types";
import { num, scoreColor, scoreLabel } from "@/lib/format";

// --- Small presentational helpers -------------------------------------------

function ScoreRing({ score, size = 116 }: { score: number; size?: number }) {
  const stroke = 10;
  const r = (size - stroke) / 2;
  const c = 2 * Math.PI * r;
  const pct = Math.max(0, Math.min(100, score)) / 100;
  const color = scoreColor(score);
  return (
    <div className="relative" style={{ width: size, height: size }}>
      <svg width={size} height={size} className="-rotate-90">
        <circle cx={size / 2} cy={size / 2} r={r} stroke="#1e293b" strokeWidth={stroke} fill="none" />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={r}
          stroke={color}
          strokeWidth={stroke}
          fill="none"
          strokeLinecap="round"
          strokeDasharray={c}
          strokeDashoffset={c * (1 - pct)}
          style={{ transition: "stroke-dashoffset 0.6s ease, stroke 0.4s ease" }}
        />
      </svg>
      <div className="absolute inset-0 flex flex-col items-center justify-center">
        <span className="text-3xl font-bold text-slate-50">{Math.round(score)}</span>
        <span className="text-[10px] uppercase tracking-wider" style={{ color }}>
          {scoreLabel(score)}
        </span>
      </div>
    </div>
  );
}

function Meter({
  label,
  value,
  invert = false,
  hint,
}: {
  label: string;
  value: number;
  invert?: boolean; // when true, higher value is worse (e.g. occlusion)
  hint?: string;
}) {
  const goodness = invert ? 100 - value : value;
  const color = scoreColor(goodness);
  return (
    <div>
      <div className="mb-1 flex items-baseline justify-between text-xs">
        <span className="text-slate-400">{label}</span>
        <span className="font-semibold" style={{ color }}>
          {Math.round(value)}
          <span className="text-slate-600">/100</span>
        </span>
      </div>
      <div className="h-2 w-full overflow-hidden rounded-full bg-slate-800">
        <div
          className="h-full rounded-full"
          style={{ width: `${Math.max(2, value)}%`, backgroundColor: color, transition: "width 0.5s ease, background-color 0.4s ease" }}
        />
      </div>
      {hint && <div className="mt-0.5 text-[10px] text-slate-500">{hint}</div>}
    </div>
  );
}

function Chip({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-slate-800 bg-slate-900/50 px-2.5 py-1.5">
      <div className="text-[10px] uppercase tracking-wide text-slate-500">{label}</div>
      <div className="text-sm font-medium text-slate-100">{value}</div>
    </div>
  );
}

const PRIORITY_STYLES: Record<string, string> = {
  high: "border-red-500/40 bg-red-500/10 text-red-300",
  medium: "border-amber-500/40 bg-amber-500/10 text-amber-300",
  low: "border-slate-600/50 bg-slate-700/20 text-slate-300",
};

// --- Main panel --------------------------------------------------------------

export default function SegmentPanel({
  detail,
  loading,
  onApplied,
}: {
  detail: SegmentDetail | null;
  loading: boolean;
  onApplied: (segment: StreetSegment) => void;
}) {
  const [recs, setRecs] = useState<Recommendation[]>([]);
  const [addedLamps, setAddedLamps] = useState(0);
  const [trim, setTrim] = useState(false);
  const [brightness, setBrightness] = useState(1);
  const [preview, setPreview] = useState<RescoreResult | null>(null);
  const [applying, setApplying] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const debounce = useRef<ReturnType<typeof setTimeout> | null>(null);

  const externalId = detail?.segment.external_id;

  // Load recommendations whenever a new segment is selected.
  useEffect(() => {
    setAddedLamps(0);
    setTrim(false);
    setBrightness(1);
    setPreview(null);
    setError(null);
    setRecs([]);
    if (!externalId) return;
    api
      .rescore(externalId, { persist: false })
      .then((r) => setRecs(r.recommendations ?? []))
      .catch((e) => setError(String(e)));
  }, [externalId]);

  const dirty = addedLamps > 0 || trim || brightness !== 1;

  // Debounced live what-if preview.
  useEffect(() => {
    if (!externalId || !dirty) {
      setPreview(null);
      return;
    }
    if (debounce.current) clearTimeout(debounce.current);
    debounce.current = setTimeout(() => {
      api
        .rescore(externalId, {
          added_lamps: addedLamps,
          trim_vegetation: trim,
          brightness_factor: brightness,
          persist: false,
        })
        .then(setPreview)
        .catch((e) => setError(String(e)));
    }, 220);
    return () => {
      if (debounce.current) clearTimeout(debounce.current);
    };
  }, [externalId, addedLamps, trim, brightness, dirty]);

  const apply = useCallback(
    async (override?: { added_lamps?: number; trim_vegetation?: boolean; brightness_factor?: number }) => {
      if (!externalId) return;
      setApplying(true);
      setError(null);
      try {
        const res = await api.rescore(externalId, {
          added_lamps: override?.added_lamps ?? addedLamps,
          trim_vegetation: override?.trim_vegetation ?? trim,
          brightness_factor: override?.brightness_factor ?? brightness,
          persist: true,
        });
        onApplied(res.segment);
        setRecs(res.recommendations ?? []);
        setAddedLamps(0);
        setTrim(false);
        setBrightness(1);
        setPreview(null);
      } catch (e) {
        setError(String(e));
      } finally {
        setApplying(false);
      }
    },
    [externalId, addedLamps, trim, brightness, onApplied],
  );

  function applyRec(rec: Recommendation) {
    if (rec.action === "install_lamps") apply({ added_lamps: Math.round(rec.params?.lamps ?? 1) });
    else if (rec.action === "trim_vegetation") apply({ trim_vegetation: true });
    else if (rec.action === "increase_brightness")
      apply({ brightness_factor: rec.params?.brightness_factor ?? 1.3 });
  }

  if (loading) {
    return <div className="h-72 animate-pulse rounded-xl bg-slate-900/60" />;
  }
  if (!detail) {
    return (
      <div className="rounded-xl border border-dashed border-slate-800 bg-slate-900/30 p-6 text-center text-sm text-slate-500">
        Click a street segment on the map to assess its lighting and plan upgrades.
      </div>
    );
  }

  const s = detail.segment;
  const current = s.overall_score;
  const projected = preview?.projected.overall_score ?? null;
  const delta = projected !== null ? projected - current : 0;

  return (
    <div className="space-y-5">
      {/* Header + overall ring */}
      <div className="flex items-center gap-4">
        <ScoreRing score={projected ?? current} />
        <div className="min-w-0 flex-1">
          <h2 className="truncate text-lg font-semibold text-slate-50">{s.name}</h2>
          <p className="text-sm text-slate-400">
            {s.district} · {s.road_type} · {num(s.length_m)} m
          </p>
          <div className="mt-2 text-xs text-slate-400">Lighting score</div>
          {projected !== null ? (
            <div className="flex items-center gap-2 text-sm">
              <span className="text-slate-500 line-through">{Math.round(current)}</span>
              <span className="text-slate-400">→</span>
              <span className="font-semibold text-slate-50">{Math.round(projected)}</span>
              <span
                className="rounded-full px-1.5 py-0.5 text-xs font-medium"
                style={{ backgroundColor: `${scoreColor(projected)}22`, color: scoreColor(projected) }}
              >
                {delta >= 0 ? "+" : ""}
                {Math.round(delta)} pts
              </span>
            </div>
          ) : (
            <div className="text-sm font-semibold text-slate-200">{Math.round(current)} / 100</div>
          )}
        </div>
      </div>

      {/* The three headline scores */}
      <div className="space-y-3 rounded-xl border border-slate-800 bg-slate-900/30 p-3">
        <Meter
          label="Lighting sufficiency"
          value={preview?.projected.lighting_sufficiency ?? s.lighting_sufficiency}
          hint={`${num(s.lighting_density, 2)} lamps/100m · target ${num(s.recommended_density, 1)}`}
        />
        <Meter
          label="Occlusion (vegetation / built mass)"
          value={preview?.projected.occlusion ?? s.occlusion}
          invert
          hint={`${s.tree_count} trees · ${Math.round(s.vegetation_ratio * 100)}% canopy`}
        />
        <Meter
          label="Infrastructure adequacy"
          value={preview?.projected.infrastructure_adequacy ?? s.infrastructure_adequacy}
          hint={`${s.street_light_count} lamps · ${s.pole_count} poles`}
        />
      </div>

      {/* Detected features */}
      <div>
        <h3 className="mb-2 text-xs font-medium uppercase tracking-wide text-slate-400">
          Detected environment
        </h3>
        <div className="grid grid-cols-3 gap-2">
          <Chip label="Lamps" value={num(s.street_light_count)} />
          <Chip label="Trees" value={num(s.tree_count)} />
          <Chip label="Vegetation" value={`${Math.round(s.vegetation_ratio * 100)}%`} />
          <Chip label="Buildings" value={`${Math.round(s.building_ratio * 100)}%`} />
          <Chip label="Sidewalk" value={`${Math.round(s.sidewalk_ratio * 100)}%`} />
          <Chip label="Road width" value={`${num(s.road_width_m, 0)} m`} />
        </div>
      </div>

      {/* Recommendations */}
      {recs.length > 0 && (
        <div>
          <h3 className="mb-2 text-xs font-medium uppercase tracking-wide text-slate-400">
            Recommended actions
          </h3>
          <div className="space-y-2">
            {recs.map((rec) => (
              <div
                key={rec.action}
                className="rounded-xl border border-slate-800 bg-slate-900/40 p-3"
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium text-slate-100">{rec.title}</span>
                      <span
                        className={`rounded-full border px-1.5 py-0.5 text-[10px] uppercase ${
                          PRIORITY_STYLES[rec.priority] ?? PRIORITY_STYLES.low
                        }`}
                      >
                        {rec.priority}
                      </span>
                    </div>
                    <p className="mt-0.5 text-xs leading-relaxed text-slate-400">{rec.detail}</p>
                  </div>
                  {rec.action !== "schedule_inspection" && (
                    <button
                      onClick={() => applyRec(rec)}
                      disabled={applying}
                      className="shrink-0 rounded-lg bg-amber-400 px-3 py-1.5 text-xs font-semibold text-slate-950 transition hover:bg-amber-300 disabled:opacity-50"
                    >
                      Apply
                    </button>
                  )}
                </div>
                {rec.projected_delta > 0 && (
                  <div className="mt-1.5 text-[11px] text-emerald-400">
                    Projected lighting score → {Math.round(rec.projected_overall_score)} (+
                    {Math.round(rec.projected_delta)} pts)
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Manual what-if planner */}
      <div className="rounded-xl border border-slate-800 bg-slate-900/30 p-3">
        <h3 className="mb-3 text-xs font-medium uppercase tracking-wide text-slate-400">
          Plan an intervention
        </h3>

        <div className="space-y-3">
          <div>
            <div className="mb-1 flex justify-between text-xs text-slate-400">
              <span>Install lamp posts</span>
              <span className="font-medium text-slate-200">+{addedLamps}</span>
            </div>
            <input
              type="range"
              min={0}
              max={12}
              value={addedLamps}
              onChange={(e) => setAddedLamps(Number(e.target.value))}
              className="w-full accent-amber-400"
            />
          </div>

          <div>
            <div className="mb-1 flex justify-between text-xs text-slate-400">
              <span>Lamp brightness</span>
              <span className="font-medium text-slate-200">{Math.round(brightness * 100)}%</span>
            </div>
            <input
              type="range"
              min={100}
              max={160}
              step={5}
              value={brightness * 100}
              onChange={(e) => setBrightness(Number(e.target.value) / 100)}
              className="w-full accent-amber-400"
            />
          </div>

          <label className="flex items-center gap-2 text-sm text-slate-300">
            <input
              type="checkbox"
              checked={trim}
              onChange={(e) => setTrim(e.target.checked)}
              className="h-4 w-4 accent-emerald-400"
            />
            Trim vegetation blocking light
          </label>
        </div>

        <button
          onClick={() => apply()}
          disabled={!dirty || applying}
          className="mt-4 w-full rounded-lg bg-emerald-500 px-4 py-2 text-sm font-semibold text-slate-950 transition hover:bg-emerald-400 disabled:cursor-not-allowed disabled:opacity-40"
        >
          {applying
            ? "Applying…"
            : dirty
              ? `Apply & update map${projected !== null ? ` (→ ${Math.round(projected)})` : ""}`
              : "Adjust the controls to plan an upgrade"}
        </button>
      </div>

      {error && (
        <div className="rounded-lg border border-red-900 bg-red-950/50 px-3 py-2 text-sm text-red-300">
          {error}
        </div>
      )}

      <p className="rounded-lg bg-slate-900/60 px-3 py-2 text-[11px] leading-relaxed text-slate-500">
        KVKK: only urban assets are detected. Faces and license plates in source frames are
        irreversibly blurred before storage. No raw imagery is retained.
      </p>
    </div>
  );
}
