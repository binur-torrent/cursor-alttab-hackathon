import type { RiskLevel } from "./types";

export const colors = {
  bg: "#0b1120",
  card: "#0f172a",
  border: "#1e293b",
  text: "#e2e8f0",
  textDim: "#94a3b8",
  accent: "#fbbf24",
};

export const RISK_COLORS: Record<RiskLevel, string> = {
  low: "#22c55e",
  medium: "#eab308",
  high: "#f97316",
  critical: "#ef4444",
};

export function riskColor(level: string): string {
  return RISK_COLORS[level as RiskLevel] ?? "#94a3b8";
}
