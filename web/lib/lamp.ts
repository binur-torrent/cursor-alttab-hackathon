import { API_BASE } from "./api";

// Shared control logic + types for the Smart Lamp Prototype demo.

export const LUMINANCE_THRESHOLD = 20;
export const SEND_INTERVAL_MS = 500;

export type LampState = "ON" | "OFF";

export interface LampMessage {
  luminance: number; // 0..100 scene brightness score
  lamp: LampState;
  threshold: number;
  ts: number; // epoch ms
}

export function lampForLuminance(luminance: number): LampState {
  return luminance < LUMINANCE_THRESHOLD ? "ON" : "OFF";
}

// Resolve the relay WebSocket URL. Order of preference:
//  1. NEXT_PUBLIC_WS_URL (set this to a wss:// tunnel for a live two-phone demo)
//  2. Derived from NEXT_PUBLIC_API_URL / API_BASE (http -> ws, https -> wss)
export function lampWsUrl(room: string, role: "sensor" | "screen"): string {
  const qs = `room=${encodeURIComponent(room)}&role=${role}`;
  const explicit = process.env.NEXT_PUBLIC_WS_URL;
  const base = explicit
    ? explicit.replace(/\/$/, "")
    : `${API_BASE.replace(/^http/, "ws")}/ws/lamp`;
  return `${base}?${qs}`;
}
