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
    detector_backend: str
    faces_blurred: int
    plates_blurred: int
    anonymized: bool
    risk_score: float
    risk_level: str
    adequacy: float
    lighting_density: float
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


def _analyze_bytes(data: bytes, road_type: str, length_m: float, is_night: bool) -> AnalyzeResponse:
    anon_bytes, anon = anonymize_bytes(data)

    detector = get_detector()
    try:
        import io

        from PIL import Image  # type: ignore

        det = detector.detect_pil(Image.open(io.BytesIO(anon_bytes)).convert("RGB"))
    except Exception:
        det = detector._heuristic_from_seed(str(len(data)))  # type: ignore

    breakdown = scoring.score_segment(
        streetlights=det.street_light_count,
        length_m=length_m,
        road_type=road_type,
        night_sample_ratio=1.0 if is_night else 0.0,
    )

    return AnalyzeResponse(
        street_light_count=det.street_light_count,
        pole_count=det.pole_count,
        detector_backend=det.backend,
        faces_blurred=anon.faces_blurred,
        plates_blurred=anon.plates_blurred,
        anonymized=anon.method not in ("none", "unavailable", "read-error"),
        risk_score=breakdown.risk_score,
        risk_level=breakdown.risk_level,
        adequacy=breakdown.adequacy,
        lighting_density=breakdown.density,
        image_base64=base64.b64encode(anon_bytes).decode() if anon_bytes else None,
    )


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
        breakdown = scoring.score_segment(
            streetlights=det.street_light_count,
            length_m=req.length_m,
            road_type=req.road_type,
            night_sample_ratio=1.0 if req.is_night else 0.0,
        )
        return AnalyzeResponse(
            street_light_count=det.street_light_count,
            pole_count=det.pole_count,
            detector_backend=det.backend + "+no-streetview",
            faces_blurred=0,
            plates_blurred=0,
            anonymized=True,
            risk_score=breakdown.risk_score,
            risk_level=breakdown.risk_level,
            adequacy=breakdown.adequacy,
            lighting_density=breakdown.density,
            image_base64=None,
        )
    return _analyze_bytes(data, req.road_type, req.length_m, req.is_night)
