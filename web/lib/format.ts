import type { RiskLevel } from "./types";

// Risk color palette (also used by the map polylines).
export const RISK_COLORS: Record<RiskLevel, string> = {
  low: "#22c55e", // green
  medium: "#eab308", // amber
  high: "#f97316", // orange
  critical: "#ef4444", // red
};

export const RISK_LABEL: Record<RiskLevel, string> = {
  low: "Low",
  medium: "Medium",
  high: "High",
  critical: "Critical",
};

export function riskColor(level: string): string {
  return RISK_COLORS[(level as RiskLevel)] ?? "#94a3b8";
}

export function num(n: number, digits = 0): string {
  return n.toLocaleString("en-US", {
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  });
}

export function pct(n: number): string {
  return `${n > 0 ? "+" : ""}${num(n, 1)}%`;
}

// scoreColor maps a 0..100 "higher is better" score to a traffic-light color.
export function scoreColor(score: number): string {
  if (score >= 70) return "#22c55e"; // green
  if (score >= 45) return "#eab308"; // amber
  if (score >= 25) return "#f97316"; // orange
  return "#ef4444"; // red
}

export function scoreLabel(score: number): string {
  if (score >= 70) return "Good";
  if (score >= 45) return "Fair";
  if (score >= 25) return "Poor";
  return "Critical";
}
