"""Streetlight / pole detection from imagery.

Primary backend: `facebook/mask2former-swin-large-mapillary-vistas-panoptic`.
The Mapillary Vistas label space includes the classes we care about:

    44 = Street Light
    45 = Pole
    47 = Utility Pole

Panoptic segmentation returns one segment per *instance*, so we can count
fixtures (not just pixels). We only keep the urban-asset classes above - the
project never detects or counts people/vehicles for any purpose other than
anonymization (handled separately in `anonymize.py`).

If `transformers`/`torch` are not installed (e.g. a lightweight CI box), the
detector falls back to a deterministic heuristic so the rest of the pipeline and
the live worker still function for the demo.
"""

from __future__ import annotations

import hashlib
from dataclasses import dataclass, field
from typing import Dict, List, Optional

MODEL_ID = "facebook/mask2former-swin-large-mapillary-vistas-panoptic"

# Mapillary Vistas class ids -> our normalized fixture types.
STREET_LIGHT_ID = 44
POLE_ID = 45
UTILITY_POLE_ID = 47

URBAN_ASSET_LABELS = {
    STREET_LIGHT_ID: "street_light",
    POLE_ID: "pole",
    UTILITY_POLE_ID: "utility_pole",
}


@dataclass
class Detection:
    label: str
    score: float
    bbox: Optional[List[int]] = None  # x, y, w, h (panoptic: optional)


@dataclass
class DetectionResult:
    street_light_count: int = 0
    pole_count: int = 0
    tree_count: int = 0
    # Scene-composition ratios (0..1) — aggregate, never per-person attributes.
    vegetation_ratio: float = 0.0
    building_ratio: float = 0.0
    sidewalk_ratio: float = 0.0
    sky_ratio: float = 0.0
    detections: List[Detection] = field(default_factory=list)
    backend: str = "heuristic"

    def as_dict(self) -> Dict:
        return {
            "street_light_count": self.street_light_count,
            "pole_count": self.pole_count,
            "tree_count": self.tree_count,
            "vegetation_ratio": round(self.vegetation_ratio, 3),
            "building_ratio": round(self.building_ratio, 3),
            "sidewalk_ratio": round(self.sidewalk_ratio, 3),
            "sky_ratio": round(self.sky_ratio, 3),
            "backend": self.backend,
            "detections": [
                {"label": d.label, "score": round(d.score, 3), "bbox": d.bbox}
                for d in self.detections
            ],
        }


class StreetlightDetector:
    def __init__(self, backend: str = "mask2former"):
        self.requested_backend = backend
        self._model = None
        self._processor = None
        self._torch = None
        if backend == "mask2former":
            self._try_load_model()

    def _try_load_model(self) -> None:
        try:
            import torch  # type: ignore
            from transformers import (  # type: ignore
                AutoImageProcessor,
                Mask2FormerForUniversalSegmentation,
            )

            self._torch = torch
            self._processor = AutoImageProcessor.from_pretrained(MODEL_ID)
            self._model = Mask2FormerForUniversalSegmentation.from_pretrained(MODEL_ID)
            self._model.eval()
        except Exception as exc:  # pragma: no cover - heavy optional deps
            print(f"[detect] Mask2Former unavailable ({exc}); using heuristic backend")
            self._model = None

    @property
    def backend(self) -> str:
        return "mask2former" if self._model is not None else "heuristic"

    def detect_path(self, image_path: str) -> DetectionResult:
        if self._model is None:
            return self._heuristic(image_path)
        try:
            from PIL import Image  # type: ignore

            image = Image.open(image_path).convert("RGB")
            return self._detect_pil(image)
        except Exception as exc:  # pragma: no cover
            print(f"[detect] inference failed ({exc}); heuristic fallback")
            return self._heuristic(image_path)

    def detect_pil(self, image) -> DetectionResult:
        if self._model is None:
            return self._heuristic_from_seed(str(image.size))
        return self._detect_pil(image)

    def _detect_pil(self, image) -> DetectionResult:
        torch = self._torch
        inputs = self._processor(images=image, return_tensors="pt")
        with torch.no_grad():
            outputs = self._model(**inputs)
        processed = self._processor.post_process_panoptic_segmentation(
            outputs, target_sizes=[image.size[::-1]]
        )[0]

        result = DetectionResult(backend="mask2former")
        segmentation = processed.get("segmentation")
        id2label = getattr(self._model.config, "id2label", {}) or {}

        # Per-segment id -> pixel area, for scene-composition ratios.
        seg_areas: Dict[int, int] = {}
        total_px = 0
        if segmentation is not None:
            flat = segmentation.flatten()
            total_px = int(flat.numel())
            uniq, counts = torch.unique(flat, return_counts=True)
            seg_areas = {int(u): int(c) for u, c in zip(uniq.tolist(), counts.tolist())}

        veg_px = build_px = side_px = sky_px = 0
        for segment in processed.get("segments_info", []):
            label_id = segment.get("label_id")
            seg_id = int(segment.get("id", -1))
            area = seg_areas.get(seg_id, 0)
            name = str(id2label.get(label_id, "")).lower()

            if label_id in URBAN_ASSET_LABELS:
                label = URBAN_ASSET_LABELS[label_id]
                score = float(segment.get("score", 1.0))
                result.detections.append(Detection(label=label, score=score))
                if label == "street_light":
                    result.street_light_count += 1
                else:
                    result.pole_count += 1

            # Scene composition by class name (robust to label-id shifts).
            if "vegetation" in name or "tree" in name:
                veg_px += area
            elif "building" in name:
                build_px += area
            elif "sidewalk" in name or "curb" in name:
                side_px += area
            elif "sky" in name:
                sky_px += area

        if total_px > 0:
            result.vegetation_ratio = veg_px / total_px
            result.building_ratio = build_px / total_px
            result.sidewalk_ratio = side_px / total_px
            result.sky_ratio = sky_px / total_px
        result.tree_count = int(round(result.vegetation_ratio * 12))
        return result

    # --- Deterministic heuristic fallback (no model weights needed) ---
    def _heuristic(self, image_path: str) -> DetectionResult:
        return self._heuristic_from_seed(image_path)

    def _heuristic_from_seed(self, seed: str) -> DetectionResult:
        h = int(hashlib.sha256(seed.encode()).hexdigest(), 16)
        street_lights = h % 4  # 0..3 fixtures per frame
        poles = (h // 7) % 5
        result = DetectionResult(backend="heuristic")
        result.street_light_count = street_lights
        result.pole_count = poles + street_lights

        # Deterministic, plausible scene composition derived from the same seed.
        veg = ((h // 11) % 65) / 100.0  # 0 .. 0.64
        building = 0.15 + ((h // 13) % 55) / 100.0  # 0.15 .. 0.69
        sidewalk = ((h // 17) % 85) / 100.0
        sky = max(0.08, min(0.95, 1.0 - veg * 0.6 - building * 0.4))
        result.vegetation_ratio = veg
        result.building_ratio = building
        result.sidewalk_ratio = sidewalk
        result.sky_ratio = sky
        result.tree_count = int(round(veg * 12))

        for i in range(street_lights):
            result.detections.append(Detection(label="street_light", score=0.6 + (i % 3) * 0.1))
        for i in range(poles):
            result.detections.append(Detection(label="pole", score=0.55 + (i % 3) * 0.1))
        return result
