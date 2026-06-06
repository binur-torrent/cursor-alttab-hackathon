"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { LUMINANCE_THRESHOLD, SEND_INTERVAL_MS } from "@/lib/lamp";

export default function LampHome() {
  const [room, setRoom] = useState("demo");
  const [origin, setOrigin] = useState("");

  useEffect(() => {
    const r = new URLSearchParams(window.location.search).get("room");
    if (r) setRoom(r);
    setOrigin(window.location.origin);
  }, []);

  const sensorPath = `/lamp/sensor?room=${encodeURIComponent(room)}`;
  const screenPath = `/lamp/screen?room=${encodeURIComponent(room)}`;
  const qr = (path: string) =>
    `https://api.qrserver.com/v1/create-qr-code/?size=180x180&data=${encodeURIComponent(
      origin + path,
    )}`;

  return (
    <main className="mx-auto flex min-h-screen max-w-3xl flex-col items-center gap-6 p-6">
      <header className="text-center">
        <h1 className="flex items-center justify-center gap-2 text-2xl font-bold">
          <span className="inline-block h-3.5 w-3.5 rounded-full bg-amber-400 shadow-[0_0_14px_3px_rgba(251,191,36,0.7)]" />
          Smart Lamp Prototype
        </h1>
        <p className="mt-2 text-sm text-slate-400">
          Two phones, one street lamp. Phone A reads the light, Phone B becomes the lamp — live over
          WebSocket.
        </p>
      </header>

      <div className="grid w-full grid-cols-1 gap-4 sm:grid-cols-2">
        <Card
          who="Phone A"
          title="Sensor"
          desc="Points its camera at the scene and streams a 0–100 luminance score every 500ms."
          qr={origin ? qr(sensorPath) : ""}
          href={sensorPath}
          cta="Open sensor"
          url={origin + sensorPath}
        />
        <Card
          who="Phone B"
          title="Lamp screen"
          desc="Goes fullscreen white (max brightness) when it's dark, off when it's bright."
          qr={origin ? qr(screenPath) : ""}
          href={screenPath}
          cta="Open lamp"
          url={origin + screenPath}
        />
      </div>

      <div className="w-full rounded-2xl border border-slate-800 bg-slate-900/40 p-5">
        <strong className="text-sm">Control logic</strong>
        <p className="mt-2 text-sm leading-7 text-slate-400">
          if{" "}
          <code className="rounded bg-slate-950 px-1.5 py-0.5 text-amber-300">
            luminance &lt; {LUMINANCE_THRESHOLD}
          </code>{" "}
          → lamp <strong>ON</strong>, screen brightness <em>max</em>
          <br />
          if{" "}
          <code className="rounded bg-slate-950 px-1.5 py-0.5 text-amber-300">
            luminance &gt;= {LUMINANCE_THRESHOLD}
          </code>{" "}
          → lamp <strong>OFF</strong>, screen brightness <em>min</em>
          <br />
          Updates stream every {SEND_INTERVAL_MS}ms. Open both on the same{" "}
          <code className="rounded bg-slate-950 px-1.5 py-0.5 text-amber-300">?room={room}</code>.
        </p>
      </div>
    </main>
  );
}

function Card({
  who,
  title,
  desc,
  qr,
  href,
  cta,
  url,
}: {
  who: string;
  title: string;
  desc: string;
  qr: string;
  href: string;
  cta: string;
  url: string;
}) {
  return (
    <div className="rounded-2xl border border-slate-800 bg-slate-900/40 p-5 text-center">
      <div className="text-xs uppercase tracking-widest text-slate-500">{who}</div>
      <h2 className="text-lg font-bold">{title}</h2>
      <p className="mt-1 text-sm text-slate-400">{desc}</p>
      {qr && (
        <div className="my-3 inline-block rounded-xl bg-white p-2.5">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src={qr} alt={`${title} QR`} width={180} height={180} className="block" />
        </div>
      )}
      <div>
        <Link
          href={href}
          className="inline-block rounded-lg bg-amber-400 px-4 py-3 font-bold text-slate-950"
        >
          {cta}
        </Link>
      </div>
      <div className="mt-2 break-all text-[11px] text-slate-600">{url}</div>
    </div>
  );
}
