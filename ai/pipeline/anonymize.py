"""KVKK-compliant irreversible anonymization.

Detects faces and license plates and replaces those regions with an irreversible
Gaussian blur (the original pixels are destroyed in the output image; we never
store an un-blurred copy). This module deliberately performs **no** recognition,
identification, or tracking - it only locates regions to destroy.

Detection backends, in order of preference:
  1. OpenCV Haar cascades (faces) + a plate-shaped contour heuristic (plates).
  2. If OpenCV is unavailable, a no-op that still records that anonymization was
     requested (used only in environments without the dependency).

For production-grade face/plate detection you can swap in a HuggingFace model
(e.g. an object detector fine-tuned for faces+plates); the interface is the same:
return a list of (x, y, w, h) boxes and we blur them.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import List, Tuple

try:  # OpenCV is optional at import time so the rest of the pipeline can load.
    import cv2  # type: ignore
    import numpy as np  # type: ignore

    _HAS_CV2 = True
except Exception:  # pragma: no cover - environment without cv2
    _HAS_CV2 = False


Box = Tuple[int, int, int, int]  # x, y, w, h


@dataclass
class AnonymizationResult:
    faces_blurred: int = 0
    plates_blurred: int = 0
    boxes: List[Box] = field(default_factory=list)
    method: str = "none"

    @property
    def total(self) -> int:
        return self.faces_blurred + self.plates_blurred


def _blur_region(image, box: Box) -> None:
    x, y, w, h = box
    h_img, w_img = image.shape[:2]
    x0, y0 = max(0, x), max(0, y)
    x1, y1 = min(w_img, x + w), min(h_img, y + h)
    if x1 <= x0 or y1 <= y0:
        return
    roi = image[y0:y1, x0:x1]
    # Kernel scaled to region size; large enough to be irreversible.
    k = max(15, (min(x1 - x0, y1 - y0) // 2) * 2 + 1)
    blurred = cv2.GaussianBlur(roi, (k, k), 0)
    # Two passes + pixelation make the blur irreversible (no high-freq detail left).
    small = cv2.resize(blurred, (max(1, (x1 - x0) // 12), max(1, (y1 - y0) // 12)))
    pixelated = cv2.resize(small, (x1 - x0, y1 - y0), interpolation=cv2.INTER_NEAREST)
    image[y0:y1, x0:x1] = pixelated


def _detect_faces(gray) -> List[Box]:
    cascade_path = cv2.data.haarcascades + "haarcascade_frontalface_default.xml"
    cascade = cv2.CascadeClassifier(cascade_path)
    if cascade.empty():
        return []
    faces = cascade.detectMultiScale(gray, scaleFactor=1.1, minNeighbors=5, minSize=(24, 24))
    return [tuple(int(v) for v in f) for f in faces]


def _detect_plates(gray) -> List[Box]:
    """Heuristic plate finder: bright rectangular regions with plate-like aspect.

    This is intentionally recall-oriented (we would rather blur a non-plate than
    miss a plate). It does NOT read or recognize plates.
    """
    boxes: List[Box] = []
    edges = cv2.Canny(gray, 100, 200)
    edges = cv2.dilate(edges, np.ones((3, 3), np.uint8), iterations=1)
    contours, _ = cv2.findContours(edges, cv2.RETR_EXTERNAL, cv2.CHAIN_APPROX_SIMPLE)
    for c in contours:
        x, y, w, h = cv2.boundingRect(c)
        if w < 25 or h < 8:
            continue
        aspect = w / float(h)
        if 2.0 <= aspect <= 6.5 and (w * h) < 0.05 * gray.size:
            boxes.append((x, y, w, h))
    return boxes


def anonymize_image(image_path: str, output_path: str) -> AnonymizationResult:
    """Blur faces + plates in `image_path`, write the result to `output_path`.

    Returns counts of what was blurred. The output never contains the original
    face/plate pixels.
    """
    if not _HAS_CV2:
        return AnonymizationResult(method="unavailable")

    image = cv2.imread(image_path)
    if image is None:
        return AnonymizationResult(method="read-error")
    return _anonymize_array(image, output_path)


def anonymize_bytes(data: bytes) -> Tuple[bytes, AnonymizationResult]:
    """Anonymize an in-memory image (used by the live worker). Returns JPEG bytes."""
    if not _HAS_CV2:
        return data, AnonymizationResult(method="unavailable")
    arr = np.frombuffer(data, dtype=np.uint8)
    image = cv2.imdecode(arr, cv2.IMREAD_COLOR)
    if image is None:
        return data, AnonymizationResult(method="read-error")
    result = _anonymize_array(image, output_path=None)
    ok, buf = cv2.imencode(".jpg", image)
    return (buf.tobytes() if ok else data), result


def _anonymize_array(image, output_path: str | None) -> AnonymizationResult:
    gray = cv2.cvtColor(image, cv2.COLOR_BGR2GRAY)
    faces = _detect_faces(gray)
    plates = _detect_plates(gray)

    for box in faces:
        _blur_region(image, box)
    for box in plates:
        _blur_region(image, box)

    if output_path:
        cv2.imwrite(output_path, image)

    return AnonymizationResult(
        faces_blurred=len(faces),
        plates_blurred=len(plates),
        boxes=list(faces) + list(plates),
        method="opencv-haar+contour",
    )
