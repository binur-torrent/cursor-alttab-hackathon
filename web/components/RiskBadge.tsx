import { riskColor, RISK_LABEL } from "@/lib/format";
import type { RiskLevel } from "@/lib/types";

export default function RiskBadge({ level, score }: { level: string; score?: number }) {
  const color = riskColor(level);
  return (
    <span
      className="inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium"
      style={{ backgroundColor: `${color}22`, color }}
    >
      <span className="h-2 w-2 rounded-full" style={{ backgroundColor: color }} />
      {RISK_LABEL[level as RiskLevel] ?? level}
      {score !== undefined && <span className="opacity-80">· {score}</span>}
    </span>
  );
}
