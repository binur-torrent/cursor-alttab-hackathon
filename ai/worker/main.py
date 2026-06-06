"""LumiCity AI live-analysis worker.

A small FastAPI service the Go backend calls for on-demand analysis of a point in
Istanbul. Given either an uploaded image or a {lat, lon, heading}, it:

  1. (for coordinates) fetches a Google Street View Static image,
  2. irreversibly blurs faces + license plates (KVKK),
  3. detects streetlight fixtures with the Mapillary-Vistas Mask2Former model,
  4. scores the point with the shared scoring model,
  5. returns counts + risk + the anonymized image (base64).

Set MODEL_BACKEND=heuristic to run without downloading model weights (fast,
deploy-friendly); set MODEL_BACKEND=mask2former for real inference.
"""

from __future__ import annotations

import base64
import os
from typing import Optional

from fastapi import FastAPI, File, UploadFile
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel

from pipeline import scoring
from pipeline.anonymize import anonymize_bytes
from pipeline.detect import StreetlightDetector

app = FastAPI(title="LumiCity AI Worker", version="0.1.0")
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)

MODEL_BACKEND = os.getenv("MODEL_BACKEND", "heuristic")
GOOGLE_MAPS_API_KEY = os.getenv("GOOGLE_MAPS_API_KEY", "")

_detector: Optional[StreetlightDetector] = None


def get_detector() -> StreetlightDetector:
    global _detector
    if _detector is None:
        _detector = StreetlightDetector(backend=MODEL_BACKEND)
    return _detector


class CoordRequest(BaseModel):
    lat: float
    lon: float
    heading: float = 0.0
    road_type: str = "secondary"
    length_m: float = 100.0
    is_night: bool = False


class AnalyzeResponse(BaseModel):
    street_light_count: int
    pole_count: int
    tree_count: int = 0
    vegetation_ratio: float = 0.0
    building_ratio: float = 0.0
    sidewalk_ratio: float = 0.0
    sky_ratio: float = 0.0
    road_width_m: float = 0.0
    detector_backend: str
    faces_blurred: int
    plates_blurred: int
    anonymized: bool
    risk_score: float
    risk_level: str
    adequacy: float
    lighting_density: float
    lighting_sufficiency: float = 0.0
    occlusion: float = 0.0
    infrastructure_adequacy: float = 0.0
    overall_score: float = 0.0
    image_base64: Optional[str] = None


@app.get("/health")
def health() -> dict:
    return {
        "status": "ok",
        "detector_backend": get_detector().backend,
        "model_backend_requested": MODEL_BACKEND,
        "streetview_configured": bool(GOOGLE_MAPS_API_KEY),
    }


def _fetch_streetview(lat: float, lon: float, heading: float) -> Optional[bytes]:
    if not GOOGLE_MAPS_API_KEY:
        return None
    try:
        import urllib.parse
        import urllib.request

        params = urllib.parse.urlencode({
            "size": "640x640",
            "location": f"{lat},{lon}",
            "heading": heading,
            "fov": 90,
            "pitch": 0,
            "key": GOOGLE_MAPS_API_KEY,
        })
        url = f"https://maps.googleapis.com/maps/api/streetview?{params}"
        with urllib.request.urlopen(url, timeout=10) as resp:
            return resp.read()
    except Exception as exc:  # pragma: no cover
        print(f"[worker] streetview fetch failed: {exc}")
        return None


def _response_from_detection(det, anon, road_type: str, length_m: float, is_night: bool, image_b64: Optional[str]) -> AnalyzeResponse:
    width = scoring.road_width(road_type)
    breakdown = scoring.score_env(scoring.Features(
        streetlights=det.street_light_count,
        pole_count=det.pole_count,
        length_m=length_m,
        road_type=road_type,
        night_ratio=1.0 if is_night else 0.0,
        road_width_m=width,
        tree_count=det.tree_count,
        vegetation_ratio=det.vegetation_ratio,
        building_ratio=det.building_ratio,
        sidewalk_ratio=det.sidewalk_ratio,
        sky_ratio=det.sky_ratio,
    ))
    return AnalyzeResponse(
        street_light_count=det.street_light_count,
        pole_count=det.pole_count,
        tree_count=det.tree_count,
        vegetation_ratio=round(det.vegetation_ratio, 3),
        building_ratio=round(det.building_ratio, 3),
        sidewalk_ratio=round(det.sidewalk_ratio, 3),
        sky_ratio=round(det.sky_ratio, 3),
        road_width_m=round(width, 1),
        detector_backend=det.backend,
        faces_blurred=anon.faces_blurred if anon else 0,
        plates_blurred=anon.plates_blurred if anon else 0,
        anonymized=(anon.method not in ("none", "unavailable", "read-error")) if anon else True,
        risk_score=breakdown.risk_score,
        risk_level=breakdown.risk_level,
        adequacy=breakdown.adequacy,
        lighting_density=breakdown.density,
        lighting_sufficiency=breakdown.lighting_sufficiency,
        occlusion=breakdown.occlusion,
        infrastructure_adequacy=breakdown.infrastructure_adequacy,
        overall_score=breakdown.overall_score,
        image_base64=image_b64,
    )


def _analyze_bytes(data: bytes, road_type: str, length_m: float, is_night: bool) -> AnalyzeResponse:
    anon_bytes, anon = anonymize_bytes(data)

    detector = get_detector()
    try:
        import io

        from PIL import Image  # type: ignore

        det = detector.detect_pil(Image.open(io.BytesIO(anon_bytes)).convert("RGB"))
    except Exception:
        det = detector._heuristic_from_seed(str(len(data)))  # type: ignore

    image_b64 = base64.b64encode(anon_bytes).decode() if anon_bytes else None
    return _response_from_detection(det, anon, road_type, length_m, is_night, image_b64)


@app.post("/analyze/image", response_model=AnalyzeResponse)
async def analyze_image(
    file: UploadFile = File(...),
    road_type: str = "secondary",
    length_m: float = 100.0,
    is_night: bool = False,
) -> AnalyzeResponse:
    data = await file.read()
    return _analyze_bytes(data, road_type, length_m, is_night)


@app.post("/analyze/point", response_model=AnalyzeResponse)
def analyze_point(req: CoordRequest) -> AnalyzeResponse:
    data = _fetch_streetview(req.lat, req.lon, req.heading)
    if data is None:
        # No Street View key configured: return a deterministic heuristic result
        # so the live demo still works end-to-end.
        det = get_detector()._heuristic_from_seed(f"{req.lat},{req.lon}")  # type: ignore
        resp = _response_from_detection(det, None, req.road_type, req.length_m, req.is_night, None)
        resp.detector_backend = det.backend + "+no-streetview"
        return resp
    return _analyze_bytes(data, req.road_type, req.length_m, req.is_night)
