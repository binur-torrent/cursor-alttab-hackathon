"use client";

import { MapContainer, TileLayer, Polyline, CircleMarker, Tooltip } from "react-leaflet";
import "leaflet/dist/leaflet.css";
import type { StreetSegment } from "@/lib/types";
import { riskColor } from "@/lib/format";

interface RiskMapProps {
  segments: StreetSegment[];
  selectedId?: string | null;
  onSelect: (segment: StreetSegment) => void;
  marker?: { lat: number; lon: number } | null;
}

const ISTANBUL_CENTER: [number, number] = [41.02, 29.0];

export default function RiskMap({ segments, selectedId, onSelect, marker }: RiskMapProps) {
  return (
    <MapContainer
      center={ISTANBUL_CENTER}
      zoom={11}
      scrollWheelZoom
      style={{ height: "100%", width: "100%", background: "#0b1120" }}
    >
      <TileLayer
        attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> &copy; <a href="https://carto.com/attributions">CARTO</a>'
        url="https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png"
      />
      {segments.map((seg) => {
        const isSelected = seg.external_id === selectedId;
        if (!seg.geometry || seg.geometry.length < 2) return null;
        return (
          <Polyline
            key={seg.external_id}
            positions={seg.geometry as [number, number][]}
            pathOptions={{
              color: riskColor(seg.risk_level),
              weight: isSelected ? 8 : 4,
              opacity: isSelected ? 1 : 0.8,
            }}
            eventHandlers={{ click: () => onSelect(seg) }}
          >
            <Tooltip sticky>
              <div className="text-xs">
                <strong>{seg.name}</strong>
                <br />
                {seg.district} · risk {seg.risk_score} ({seg.risk_level})
                <br />
                {seg.street_light_count} lights · {seg.length_m} m
              </div>
            </Tooltip>
          </Polyline>
        );
      })}
      {marker && (
        <CircleMarker
          center={[marker.lat, marker.lon]}
          radius={9}
          pathOptions={{ color: "#38bdf8", fillColor: "#38bdf8", fillOpacity: 0.9 }}
        />
      )}
    </MapContainer>
  );
}
