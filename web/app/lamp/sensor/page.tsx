"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import {
  LUMINANCE_THRESHOLD,
  SEND_INTERVAL_MS,
  lampForLuminance,
  lampWsUrl,
} from "@/lib/lamp";

export default function SensorPage() {
  const videoRef = useRef<HTMLVideoElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const runningRef = useRef(false);

  const [room, setRoom] = useState("demo");
  const [started, setStarted] = useState(false);
  const [connected, setConnected] = useState(false);
  const [status, setStatus] = useState("Tap start to begin");
  const [luminance, setLuminance] = useState<number | null>(null);
  const [ts, setTs] = useState<number | null>(null);

  useEffect(() => {
    const r = new URLSearchParams(window.location.search).get("room");
    if (r) setRoom(r);
  }, []);

  const connect = useCallback(
    (rm: string) => {
      const ws = new WebSocket(lampWsUrl(rm, "sensor"));
      ws.onopen = () => setConnected(true);
      ws.onclose = () => {
        setConnected(false);
        if (runningRef.current) setTimeout(() => connect(rm), 1500);
      };
      wsRef.current = ws;
    },
    [],
  );

  // Average Rec.601 luminance of a downscaled frame, mapped to 0..100.
  const estimate = useCallback((): number | null => {
    const video = videoRef.current;
    const canvas = canvasRef.current;
    if (!video || !canvas || video.readyState < 2) return null;
    const ctx = canvas.getContext("2d", { willReadFrequently: true });
    if (!ctx) return null;
    ctx.drawImage(video, 0, 0, canvas.width, canvas.height);
    const { data } = ctx.getImageData(0, 0, canvas.width, canvas.height);
    let sum = 0;
    for (let i = 0; i < data.length; i += 4) {
      sum += 0.299 * data[i] + 0.587 * data[i + 1] + 0.114 * data[i + 2];
    }
    const avg = sum / (data.length / 4); // 0..255
    return Math.round((avg / 255) * 100);
  }, []);

  const start = useCallback(async () => {
    if (started) return;
    setStatus("Requesting camera…");
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: { ideal: "environment" } },
        audio: false,
      });
      const video = videoRef.current!;
      video.srcObject = stream;
      await video.play();
      runningRef.current = true;
      setStarted(true);
      setStatus(`Streaming every ${SEND_INTERVAL_MS}ms`);
      connect(room);

      const timer = setInterval(() => {
        const lum = estimate();
        if (lum == null) return;
        const now = Date.now();
        setLuminance(lum);
        setTs(now);
        const ws = wsRef.current;
        if (ws && ws.readyState === WebSocket.OPEN) {
          ws.send(
            JSON.stringify({
              luminance: lum,
              lamp: lampForLuminance(lum),
              threshold: LUMINANCE_THRESHOLD,
              ts: now,
            }),
          );
        }
      }, SEND_INTERVAL_MS);
      return () => clearInterval(timer);
    } catch (e) {
      setStatus(`Camera blocked: ${e instanceof Error ? e.message : e} (needs HTTPS)`);
    }
  }, [started, room, connect, estimate]);

  useEffect(() => {
    return () => {
      runningRef.current = false;
      wsRef.current?.close();
    };
  }, []);

  const lampOn = luminance != null && luminance < LUMINANCE_THRESHOLD;

  return (
    <main className="mx-auto flex min-h-screen max-w-md flex-col gap-4 p-4">
      <header className="flex items-center gap-2">
        <span className="inline-block h-3 w-3 rounded-full bg-amber-400 shadow-[0_0_12px_2px_rgba(251,191,36,0.7)]" />
        <div>
          <h1 className="text-lg font-bold">Sensor · Phone A</h1>
          <p className="text-xs text-slate-400">Estimates scene luminance from the camera</p>
        </div>
      </header>

      <video
        ref={videoRef}
        playsInline
        muted
        className="aspect-[4/3] w-full rounded-2xl border border-slate-800 bg-black object-cover"
      />
      <canvas ref={canvasRef} width={64} height={48} className="hidden" />

      <div className="rounded-2xl border border-slate-800 bg-slate-900/40 p-4">
        <div className="text-6xl font-extrabold leading-none">
          {luminance ?? "--"}
          <span className="text-lg font-semibold text-slate-400"> / 100</span>
        </div>
        <div className="mt-3 h-3 overflow-hidden rounded-full bg-slate-800">
          <div
            className="h-full bg-gradient-to-r from-sky-500 to-amber-400 transition-all"
            style={{ width: `${luminance ?? 0}%` }}
          />
        </div>
        <div className="mt-3 flex items-center justify-between text-sm text-slate-400">
          <span>
            Lamp would be{" "}
            <span
              className={`rounded-full px-2 py-0.5 text-xs font-bold ${
                lampOn
                  ? "border border-amber-400 bg-amber-400/15 text-amber-300"
                  : "border border-slate-600 bg-slate-600/15 text-slate-400"
              }`}
            >
              {lampOn ? "ON" : "OFF"}
            </span>
          </span>
          <span>{ts ? new Date(ts).toLocaleTimeString() : "—"}</span>
        </div>
      </div>

      <button
        onClick={start}
        disabled={started}
        className="rounded-xl bg-amber-400 py-4 text-base font-bold text-slate-950 disabled:opacity-60"
      >
        {started ? "Streaming…" : "Start camera & streaming"}
      </button>

      <p className="text-center text-xs text-slate-400">
        <span
          className={`mr-1.5 inline-block h-2 w-2 rounded-full ${
            connected ? "bg-green-500" : "bg-slate-600"
          }`}
        />
        {status}
      </p>
      <p className="text-center text-[11px] text-slate-600">
        room: {room} · threshold: {LUMINANCE_THRESHOLD}
      </p>
    </main>
  );
}
