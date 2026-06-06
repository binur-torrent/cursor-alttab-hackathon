"use client";

import dynamic from "next/dynamic";
import type { ComponentProps } from "react";
import type RiskMap from "./RiskMap";

// Leaflet must run client-side only; load the map without SSR.
const RiskMapNoSSR = dynamic(() => import("./RiskMap"), {
  ssr: false,
  loading: () => (
    <div className="flex h-full w-full items-center justify-center bg-slate-950 text-slate-500">
      Loading map…
    </div>
  ),
});

export default function MapView(props: ComponentProps<typeof RiskMap>) {
  return <RiskMapNoSSR {...props} />;
}
