# LumiCity AI — pipeline & worker

HuggingFace-powered computer vision for urban streetlight analysis.

## What's here

| Module | Purpose |
|--------|---------|
| `pipeline/scoring.py` | Lighting-adequacy / risk model (mirrored 1:1 in Go) |
| `pipeline/geo.py` | Istanbul bounding box + ~150 m segment grid binning |
| `pipeline/anonymize.py` | KVKK: irreversible face + license-plate blur |
| `pipeline/detect.py` | Streetlight/pole detection (Mask2Former Mapillary-Vistas) |
| `pipeline/run.py` | Full ingestion: HF dataset -> anonymize -> detect -> seed JSON |
| `pipeline/generate_seed.py` | Dependency-free deterministic Istanbul seed generator |
| `worker/main.py` | FastAPI live-analysis service (Street View on demand) |

## Data source (HuggingFace)

Primary dataset: [`Reubencf/streetview-global`](https://huggingface.co/datasets/Reubencf/streetview-global)
— Mapillary-sourced street imagery with `latitude`, `longitude`, `compass_angle`,
`time_of_day`, `road_type`, `infrastructure`. We stream it and keep only frames
inside the Istanbul bounding box.

Secondary / alternative: [`nyuuzyou/streetview`](https://huggingface.co/datasets/nyuuzyou/streetview)
(coordinates from 20.49°E eastward, covers Istanbul).

Detection model: [`facebook/mask2former-swin-large-mapillary-vistas-panoptic`](https://huggingface.co/facebook/mask2former-swin-large-mapillary-vistas-panoptic).
The Mapillary Vistas label space gives us class `44 = Street Light`, `45 = Pole`,
`47 = Utility Pole`. Panoptic segmentation returns one segment per instance, so we
count fixtures, not pixels.

## Quick start

```bash
pip install -r requirements.txt   # only needed for live ingestion / worker

# 1. Regenerate the demo seed (no heavy deps, instant):
python -m pipeline.generate_seed --out ../data/seed/segments.json

# 2. Real ingestion from HuggingFace (downloads model on first run):
python -m pipeline.run --dataset Reubencf/streetview-global --limit 1000 \
    --out ../data/seed/segments.json --model-backend mask2former

# 3. Run the live worker:
uvicorn worker.main:app --reload --port 8000
#   POST /analyze/point  {"lat":41.005,"lon":28.95,"heading":90}
#   POST /analyze/image  (multipart file)
```

## Privacy

This pipeline detects **urban assets only**. It never recognizes, identifies, or
tracks people or vehicles. Faces and plates are irreversibly blurred (Gaussian +
pixelation) before any storage or detection. The committed seed file contains
**derived metadata only** — no raw imagery. See `../docs/KVKK_COMPLIANCE.md`.
