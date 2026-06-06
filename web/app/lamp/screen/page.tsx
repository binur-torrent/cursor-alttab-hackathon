"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import {
  LUMINANCE_THRESHOLD,
  lampForLuminance,
  lampWsUrl,
  type LampMessage,
} from "@/lib/lamp";

const STALE_MS = 3000;

export default function ScreenPage() {
  const wsRef = useRef<WebSocket | null>(null);
  const aliveRef = useRef(true);

  const [room, setRoom] = useState("demo");
  const [started, setStarted] = useState(false);
  const [connected, setConnected] = useState(false);
  const [luminance, setLuminance] = useState<number | null>(null);
  const [lampOn, setLampOn] = useState(false);
  const [ts, setTs] = useState<number | null>(null);
  const [stale, setStale] = useState(false);

  useEffect(() => {
    const r = new URLSearchParams(window.location.search).get("room");
    if (r) setRoom(r);
  }, []);

  const connect = useCallback((rm: string) => {
    const ws = new WebSocket(lampWsUrl(rm, "screen"));
    ws.onopen = () => setConnected(true);
    ws.onmessage = (ev) => {
      try {
        const msg = JSON.parse(ev.data) as LampMessage;
        if (typeof msg.luminance !== "number") return;
        setLuminance(msg.luminance);
        setLampOn(msg.lamp ? msg.lamp === "ON" : lampForLuminance(msg.luminance) === "ON");
        setTs(msg.ts || Date.now());
        setStale(false);
      } catch {
        /* ignore malformed frames */
      }
    };
    ws.onclose = () => {
      setConnected(false);
      if (aliveRef.current) setTimeout(() => connect(rm), 1500);
    };
    wsRef.current = ws;
  }, []);

  useEffect(() => {
    const id = setInterval(() => {
      if (ts && Date.now() - ts > STALE_MS) setStale(true);
    }, 1000);
    return () => clearInterval(id);
  }, [ts]);

  useEffect(() => {
    return () => {
      aliveRef.current = false;
      wsRef.current?.close();
    };
  }, []);

  const activate = useCallback(() => {
    setStarted(true);
    const el = document.documentElement;
    el.requestFullscreen?.().catch(() => {});
    connect(room);
  }, [room, connect]);

  return (
    <main
      className={`flex min-h-screen select-none items-center justify-center transition-colors duration-300 ${
        lampOn ? "bg-white text-slate-900" : "bg-[#05070d] text-slate-600"
      }`}
    >
      {!started ? (
        <button
          onClick={activate}
          className="rounded-2xl bg-amber-400 px-7 py-5 text-lg font-extrabold text-slate-950"
        >
          Activate lamp (fullscreen)
        </button>
      ) : (
        <div className="text-center">
          <div className="text-[clamp(48px,16vw,160px)] font-black leading-none">
            {lampOn ? "ON" : "OFF"}
          </div>
          <div className="mt-2 text-[clamp(20px,6vw,40px)] font-bold">
            {luminance ?? "--"}
            <span className="text-[0.5em] opacity-60"> / 100 lux-score</span>
          </div>
          <div className="mt-3 text-sm opacity-70">
            <span
              className={`mr-1.5 inline-block h-2 w-2 rounded-full ${
                connected ? "bg-green-500" : "bg-slate-500"
              }`}
            />
            {ts
              ? `${stale ? "sensor silent since" : "updated"} ${new Date(ts).toLocaleTimeString()}`
              : "waiting for sensor…"}
          </div>
          <div className="mt-1.5 text-xs uppercase tracking-widest opacity-55">
            room: {room} · lamp ON when luminance &lt; {LUMINANCE_THRESHOLD}
          </div>
        </div>
      )}
    </main>
  );
}
