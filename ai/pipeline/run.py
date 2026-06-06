"""LumiCity AI ingestion pipeline (HuggingFace -> anonymize -> detect -> seed).

Pulls geolocated street-level imagery from HuggingFace, keeps only frames inside
the Istanbul bounding box, irreversibly blurs faces/plates, detects streetlight
fixtures with a Mapillary-Vistas Mask2Former model, aggregates frames into
~150 m street segments, scores each segment, and writes an anonymized seed file
to `data/seed/segments.json`.

Usage:
    python -m pipeline.run --limit 500 --out ../data/seed/segments.json
    python -m pipeline.run --dataset Reubencf/streetview-global --limit 1000

Notes:
- Uses streaming mode so we never download the full multi-GB dataset.
- Only derived metadata + detection counts are written to the seed file. Raw
  images are written (already anonymized) to data/raw/ which is gitignored and
  must be deleted after the hackathon (see docs/KVKK_COMPLIANCE.md).
"""

from __future__ import annotations

import argparse
import json
import os
import sys
from collections import defaultdict
from datetime import datetime, timezone
from typing import Dict, List

from . import geo, scoring
from .detect import StreetlightDetector

# Primary HF dataset: globally-sampled Mapillary street view with lat/lon,
# compass, time_of_day, road_type. Secondary fallback also documented in README.
DEFAULT_DATASET = "Reubencf/streetview-global"

NIGHT_TIMES = {"night", "dusk"}


def _load_stream(dataset: str):
    """Yield records from a HF dataset in streaming mode."""
    try:
        from datasets import load_dataset  # type: ignore
    except Exception as exc:
        print(
            "[run] The 'datasets' package is required for live ingestion. "
            "Install with: pip install -r requirements.txt\n"
            f"       (import error: {exc})",
            file=sys.stderr,
        )
        sys.exit(2)
    return load_dataset(dataset, split="train", streaming=True)


def _road_type_from_record(rec: Dict) -> str:
    """Map dataset's free-form road descriptors to our road classes."""
    raw = (rec.get("road_type") or rec.get("highway") or "").lower()
    if raw in scoring.RECOMMENDED_DENSITY:
        return raw
    if "motor" in raw or "highway" in raw or "trunk" in raw:
        return "highway"
    if "primary" in raw or "avenue" in raw or "main" in raw:
        return "primary"
    if "residential" in raw or "living" in raw:
        return "residential"
    if "service" in raw or "alley" in raw:
        return "service"
    return scoring.DEFAULT_ROAD_TYPE


def run(dataset: str, limit: int, out_path: str, raw_dir: str, model_backend: str) -> None:
    os.makedirs(os.path.dirname(out_path), exist_ok=True)
    os.makedirs(raw_dir, exist_ok=True)

    detector = StreetlightDetector(backend=model_backend)
    print(f"[run] dataset={dataset} limit={limit} detector_backend={detector.backend}")

    stream = _load_stream(dataset)

    # Aggregate per grid cell (each cell -> one street segment).
    cells: Dict[geo.GridCell, dict] = defaultdict(
        lambda: {
            "samples": [],
            "street_lights": 0,
            "poles": 0,
            "night_samples": 0,
            "road_types": defaultdict(int),
            "lats": [],
            "lons": [],
        }
    )

    kept = 0
    seen = 0
    for rec in stream:
        seen += 1
        if seen > limit:
            break
        lat = rec.get("latitude")
        lon = rec.get("longitude")
        if lat is None or lon is None or not geo.in_istanbul(float(lat), float(lon)):
            continue

        lat, lon = float(lat), float(lon)
        from .anonymize import anonymize_bytes  # local import keeps cv2 optional

        # Anonymize the image bytes before any detection or storage.
        det_result = None
        img = rec.get("image")
        anon = None
        if img is not None:
            try:
                import io

                buf = io.BytesIO()
                img.convert("RGB").save(buf, format="JPEG")
                anon_bytes, anon = anonymize_bytes(buf.getvalue())
                # Persist anonymized frame (gitignored, deleted post-hackathon).
                raw_name = f"{rec.get('id', kept)}.anon.jpg"
                with open(os.path.join(raw_dir, raw_name), "wb") as f:
                    f.write(anon_bytes)
                from PIL import Image  # type: ignore

                det_result = detector.detect_pil(Image.open(io.BytesIO(anon_bytes)).convert("RGB"))
            except Exception as exc:
                print(f"[run] frame skipped ({exc})")
                continue
        if det_result is None:
            det_result = detector._heuristic_from_seed(f"{lat},{lon}")  # type: ignore

        road_type = _road_type_from_record(rec)
        time_of_day = (rec.get("time_of_day") or "").lower()
        is_night = time_of_day in NIGHT_TIMES

        cell = geo.cell_for(lat, lon)
        agg = cells[cell]
        agg["street_lights"] += det_result.street_light_count
        agg["poles"] += det_result.pole_count
        agg["road_types"][road_type] += 1
        agg["lats"].append(lat)
        agg["lons"].append(lon)
        if is_night:
            agg["night_samples"] += 1
        agg["samples"].append(
            {
                "id": str(rec.get("id", f"s{kept}")),
                "lat": lat,
                "lon": lon,
                "heading": rec.get("compass_angle"),
                "captured_at": rec.get("captured_at"),
                "time_of_day": time_of_day or None,
                "road_type": road_type,
                "street_light_count": det_result.street_light_count,
                "pole_count": det_result.pole_count,
                "anonymized": bool(anon and anon.total >= 0),
                "faces_blurred": anon.faces_blurred if anon else 0,
                "plates_blurred": anon.plates_blurred if anon else 0,
                "backend": det_result.backend,
            }
        )
        kept += 1

    segments = _build_segments(cells)
    payload = {
        "generated_by": "lumicity-ai/pipeline",
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "source_dataset": dataset,
        "source_license": "Mapillary CC-BY-SA (imagery); derived metadata only in this file",
        "detector_backend": detector.backend,
        "frames_seen": seen,
        "frames_in_istanbul": kept,
        "segment_count": len(segments),
        "segments": segments,
    }
    with open(out_path, "w") as f:
        json.dump(payload, f, indent=2)
    print(f"[run] wrote {len(segments)} segments to {out_path} (from {kept} Istanbul frames)")


def _build_segments(cells: Dict[geo.GridCell, dict]) -> List[dict]:
    segments = []
    for cell, agg in cells.items():
        n = len(agg["samples"])
        if n == 0:
            continue
        lat_c = sum(agg["lats"]) / n
        lon_c = sum(agg["lons"]) / n
        # Estimate segment length from the spread of sample points (min 80 m).
        length_m = max(
            80.0,
            geo.haversine_m(min(agg["lats"]), min(agg["lons"]), max(agg["lats"]), max(agg["lons"])),
        )
        road_type = max(agg["road_types"].items(), key=lambda kv: kv[1])[0]
        night_ratio = agg["night_samples"] / n
        breakdown = scoring.score_segment(
            streetlights=agg["street_lights"],
            length_m=length_m,
            road_type=road_type,
            night_sample_ratio=night_ratio,
        )
        segments.append(
            {
                "external_id": f"seg-{cell.row}-{cell.col}",
                "centroid_lat": round(lat_c, 6),
                "centroid_lon": round(lon_c, 6),
                "road_type": road_type,
                "length_m": round(length_m, 1),
                "sample_count": n,
                "street_light_count": agg["street_lights"],
                "pole_count": agg["poles"],
                "night_sample_ratio": round(night_ratio, 3),
                "lighting_density": breakdown.density,
                "recommended_density": breakdown.recommended,
                "adequacy": breakdown.adequacy,
                "risk_score": breakdown.risk_score,
                "risk_level": breakdown.risk_level,
                "samples": agg["samples"][:5],  # keep a few representative frames
            }
        )
    segments.sort(key=lambda s: s["risk_score"], reverse=True)
    return segments


def main() -> None:
    parser = argparse.ArgumentParser(description="LumiCity AI ingestion pipeline")
    parser.add_argument("--dataset", default=DEFAULT_DATASET, help="HuggingFace dataset id")
    parser.add_argument("--limit", type=int, default=500, help="max frames to scan from stream")
    parser.add_argument("--out", default="../data/seed/segments.json", help="seed output path")
    parser.add_argument("--raw-dir", default="../data/raw", help="dir for anonymized frames (gitignored)")
    parser.add_argument(
        "--model-backend",
        default="mask2former",
        choices=["mask2former", "heuristic"],
        help="detection backend",
    )
    args = parser.parse_args()
    run(args.dataset, args.limit, args.out, args.raw_dir, args.model_backend)


if __name__ == "__main__":
    main()
