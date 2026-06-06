"""Deterministic Istanbul seed generator (no heavy dependencies).

Produces `data/seed/segments.json` with the SAME schema the real HuggingFace
pipeline (`run.py`) emits, so the backend and dashboard have realistic,
geographically-correct data to serve immediately - even on a machine that can't
download multi-GB models/datasets.

The generator walks a curated set of real Istanbul corridors, splits each into
~200 m segments, and assigns streetlight counts from a per-corridor "lighting
quality" factor plus deterministic noise. It then scores every segment with the
shared `scoring` module (identical formula to the Go backend), yielding a
realistic spread of low/medium/high/critical-risk segments for the demo.

Only derived metadata is written - never raw imagery (KVKK compliant).

    python -m pipeline.generate_seed --out ../data/seed/segments.json
"""

from __future__ import annotations

import argparse
import json
import os
import random
from datetime import datetime, timezone, timedelta
from typing import List, Tuple

from . import geo, scoring

Coord = Tuple[float, float]

# Curated real Istanbul corridors: (name, district, road_type, quality, waypoints).
# `quality` in [0,1] is how well-lit the corridor actually is (drives risk spread).
CORRIDORS = [
    ("Bağdat Caddesi", "Kadıköy", "primary", 0.92, [
        (40.9760, 29.0560), (40.9680, 29.0640), (40.9620, 29.0780),
        (40.9550, 29.0920), (40.9480, 29.1050),
    ]),
    ("İstiklal Caddesi", "Beyoğlu", "primary", 0.95, [
        (41.0360, 28.9770), (41.0335, 28.9760), (41.0310, 28.9755),
        (41.0285, 28.9745), (41.0270, 28.9740),
    ]),
    ("Barbaros Bulvarı", "Beşiktaş", "primary", 0.80, [
        (41.0430, 29.0080), (41.0490, 29.0090), (41.0550, 29.0095), (41.0610, 29.0080),
    ]),
    ("Büyükdere Caddesi", "Şişli", "highway", 0.78, [
        (41.0670, 29.0150), (41.0780, 29.0130), (41.0850, 29.0110), (41.1080, 29.0220),
    ]),
    ("Kennedy Caddesi (Sahil Yolu)", "Fatih", "highway", 0.55, [
        (41.0050, 28.9500), (41.0030, 28.9400), (41.0010, 28.9300), (40.9990, 28.9200),
    ]),
    ("Vatan Caddesi", "Fatih", "primary", 0.70, [
        (41.0130, 28.9390), (41.0150, 28.9300), (41.0170, 28.9230),
    ]),
    ("Moda Caddesi", "Kadıköy", "secondary", 0.74, [
        (40.9830, 29.0260), (40.9800, 29.0270), (40.9780, 29.0290),
    ]),
    ("Bağlarbaşı Caddesi", "Üsküdar", "secondary", 0.62, [
        (41.0270, 29.0150), (41.0240, 29.0220), (41.0215, 29.0290),
    ]),
    ("D100 Anadolu", "Ümraniye", "highway", 0.66, [
        (40.9930, 29.1000), (40.9970, 29.1150), (41.0010, 29.1300),
    ]),
    ("Doğu Caddesi", "Esenyurt", "residential", 0.34, [
        (41.0290, 28.6700), (41.0330, 28.6750), (41.0360, 28.6800),
    ]),
    ("Fatih Bulvarı", "Sultanbeyli", "residential", 0.30, [
        (40.9650, 29.2670), (40.9690, 29.2720), (40.9720, 29.2780),
    ]),
    ("Bakırköy Sahil", "Bakırköy", "secondary", 0.58, [
        (40.9750, 28.8720), (40.9730, 28.8800), (40.9715, 28.8880),
    ]),
    ("Abbasağa Sokakları", "Beşiktaş", "residential", 0.48, [
        (41.0445, 29.0010), (41.0460, 28.9985), (41.0475, 28.9960),
    ]),
    ("Tarlabaşı Bulvarı", "Beyoğlu", "primary", 0.52, [
        (41.0380, 28.9760), (41.0410, 28.9740), (41.0440, 28.9720),
    ]),
    ("Alemdağ Caddesi", "Çekmeköy", "secondary", 0.40, [
        (41.0350, 29.1750), (41.0390, 29.1820), (41.0430, 29.1890),
    ]),
]

SEGMENT_LEN_M = 200.0
TIMES = ["day", "dusk", "night", "dawn"]

# Environmental feature priors by road class. These shape how much tree canopy,
# built mass, sidewalk and open sky a typical segment of each class shows in
# street-level imagery, so the occlusion / infrastructure scores vary realistically.
VEGETATION_BASE = {
    "highway": 0.12,
    "primary": 0.22,
    "secondary": 0.38,
    "residential": 0.48,
    "service": 0.42,
}
BUILDING_BASE = {
    "highway": 0.15,
    "primary": 0.50,
    "secondary": 0.55,
    "residential": 0.45,
    "service": 0.40,
}
SIDEWALK_BASE = {
    "highway": 0.25,
    "primary": 0.70,
    "secondary": 0.75,
    "residential": 0.65,
    "service": 0.45,
}


def derive_features(rng: random.Random, road_type: str, quality: float, length_m: float) -> dict:
    """Deterministically derive environmental features for a segment."""
    veg = scoring.clamp(VEGETATION_BASE.get(road_type, 0.35) * (1 + rng.uniform(-0.3, 0.45)), 0.02, 0.95)
    building = scoring.clamp(BUILDING_BASE.get(road_type, 0.5) * (1 + rng.uniform(-0.25, 0.25)), 0.05, 0.9)
    sidewalk = scoring.clamp(SIDEWALK_BASE.get(road_type, 0.6) * (0.6 + 0.5 * quality) * (1 + rng.uniform(-0.15, 0.15)), 0.0, 1.0)
    tree_count = int(round(veg * (length_m / 100.0) * rng.uniform(2.0, 5.0)))
    sky = scoring.clamp(1.0 - veg * 0.6 - building * 0.4, 0.1, 0.95)
    width = scoring.road_width(road_type) * rng.uniform(0.9, 1.15)
    return {
        "tree_count": tree_count,
        "vegetation_ratio": round(veg, 3),
        "building_ratio": round(building, 3),
        "road_width_m": round(width, 1),
        "sidewalk_ratio": round(sidewalk, 3),
        "sky_ratio": round(sky, 3),
        "brightness_factor": 1.0,
    }


def interpolate(a: Coord, b: Coord, t: float) -> Coord:
    return (a[0] + (b[0] - a[0]) * t, a[1] + (b[1] - a[1]) * t)


def split_corridor(waypoints: List[Coord]) -> List[List[Coord]]:
    """Split a polyline into ~SEGMENT_LEN_M segments, each a small polyline."""
    segments: List[List[Coord]] = []
    carry: List[Coord] = [waypoints[0]]
    carry_len = 0.0
    for i in range(len(waypoints) - 1):
        a, b = waypoints[i], waypoints[i + 1]
        seg_len = geo.haversine_m(a[0], a[1], b[0], b[1])
        steps = max(1, int(seg_len // 25))  # sample every ~25 m
        for s in range(1, steps + 1):
            p = interpolate(a, b, s / steps)
            step_len = geo.haversine_m(carry[-1][0], carry[-1][1], p[0], p[1])
            carry.append(p)
            carry_len += step_len
            if carry_len >= SEGMENT_LEN_M:
                segments.append(carry)
                carry = [p]
                carry_len = 0.0
    if len(carry) >= 2:
        segments.append(carry)
    return segments


def polyline_length(poly: List[Coord]) -> float:
    total = 0.0
    for i in range(len(poly) - 1):
        total += geo.haversine_m(poly[i][0], poly[i][1], poly[i + 1][0], poly[i + 1][1])
    return total


def build(seed: int) -> dict:
    rng = random.Random(seed)
    segments = []
    base_time = datetime(2024, 11, 1, tzinfo=timezone.utc)

    for ci, (name, district, road_type, quality, waypoints) in enumerate(CORRIDORS):
        for si, poly in enumerate(split_corridor(waypoints)):
            length_m = max(80.0, polyline_length(poly))
            target = scoring.recommended_density(road_type)
            # Actual installed density = quality * target, with noise + occasional outages.
            noise = rng.uniform(-0.25, 0.25)
            # A small share of segments are dark spots: full outage (dead lamps)
            # or a partial outage. These surface as critical/high-risk hotspots.
            roll = rng.random()
            if roll < 0.05:
                outage = 0.0  # complete dark spot
            elif roll < 0.15:
                outage = 0.5  # partial outage
            else:
                outage = 1.0
            actual_density = max(0.0, target * quality * outage * (1 + noise))
            street_lights = int(round(actual_density * (length_m / 100.0)))
            poles = street_lights + rng.randint(0, 3)

            n_samples = rng.randint(3, 8)
            night_samples = sum(1 for _ in range(n_samples) if rng.random() < 0.45)
            night_ratio = night_samples / n_samples

            features = derive_features(rng, road_type, quality, length_m)
            breakdown = scoring.score_env(scoring.Features(
                streetlights=street_lights,
                pole_count=poles,
                length_m=length_m,
                road_type=road_type,
                night_ratio=night_ratio,
                road_width_m=features["road_width_m"],
                tree_count=features["tree_count"],
                vegetation_ratio=features["vegetation_ratio"],
                building_ratio=features["building_ratio"],
                sidewalk_ratio=features["sidewalk_ratio"],
                sky_ratio=features["sky_ratio"],
                brightness_factor=features["brightness_factor"],
            ))

            centroid = poly[len(poly) // 2]
            samples = []
            for k in range(min(n_samples, 5)):
                p = poly[min(len(poly) - 1, k * (len(poly) // max(1, n_samples)))]
                tod = rng.choice(TIMES)
                samples.append({
                    "id": f"mly-{ci}{si}{k}-{rng.randint(10000, 99999)}",
                    "lat": round(p[0], 6),
                    "lon": round(p[1], 6),
                    "heading": round(rng.uniform(0, 360), 1),
                    "captured_at": (base_time + timedelta(days=rng.randint(0, 120))).isoformat(),
                    "time_of_day": tod,
                    "road_type": road_type,
                    "street_light_count": max(0, int(round(actual_density * 0.5)) + rng.randint(-1, 1)),
                    "pole_count": rng.randint(0, 2),
                    "anonymized": True,
                    "faces_blurred": rng.randint(0, 3),
                    "plates_blurred": rng.randint(0, 4),
                    "backend": "mask2former",
                })

            segments.append({
                "external_id": f"seg-{ci:02d}-{si:02d}",
                "name": f"{name} - bölüm {si + 1}",
                "district": district,
                "road_type": road_type,
                "centroid_lat": round(centroid[0], 6),
                "centroid_lon": round(centroid[1], 6),
                "geometry": [[round(p[0], 6), round(p[1], 6)] for p in poly],
                "length_m": round(length_m, 1),
                "sample_count": n_samples,
                "street_light_count": street_lights,
                "pole_count": poles,
                "night_sample_ratio": round(night_ratio, 3),
                "tree_count": features["tree_count"],
                "vegetation_ratio": features["vegetation_ratio"],
                "building_ratio": features["building_ratio"],
                "road_width_m": features["road_width_m"],
                "sidewalk_ratio": features["sidewalk_ratio"],
                "sky_ratio": features["sky_ratio"],
                "brightness_factor": features["brightness_factor"],
                "lighting_density": breakdown.density,
                "recommended_density": breakdown.recommended,
                "adequacy": breakdown.adequacy,
                "lighting_sufficiency": breakdown.lighting_sufficiency,
                "occlusion": breakdown.occlusion,
                "infrastructure_adequacy": breakdown.infrastructure_adequacy,
                "overall_score": breakdown.overall_score,
                "risk_score": breakdown.risk_score,
                "risk_level": breakdown.risk_level,
                "samples": samples,
            })

    segments.sort(key=lambda s: s["risk_score"], reverse=True)
    return {
        "generated_by": "lumicity-ai/pipeline.generate_seed",
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "source_dataset": "Reubencf/streetview-global (schema); curated Istanbul corridors",
        "source_license": "Derived metadata only - no raw imagery (KVKK compliant)",
        "detector_backend": "mask2former (schema-compatible)",
        "frames_seen": sum(s["sample_count"] for s in segments),
        "frames_in_istanbul": sum(s["sample_count"] for s in segments),
        "segment_count": len(segments),
        "segments": segments,
    }


def main() -> None:
    parser = argparse.ArgumentParser(description="Generate deterministic Istanbul seed data")
    parser.add_argument("--out", default="../data/seed/segments.json")
    parser.add_argument("--seed", type=int, default=42)
    args = parser.parse_args()

    payload = build(args.seed)
    os.makedirs(os.path.dirname(os.path.abspath(args.out)), exist_ok=True)
    with open(args.out, "w") as f:
        json.dump(payload, f, indent=2, ensure_ascii=False)
    print(f"[generate_seed] wrote {payload['segment_count']} segments to {args.out}")


if __name__ == "__main__":
    main()
