# KVKK & Privacy Compliance

LumiCity AI processes street-level imagery to assess **public lighting
infrastructure**. This document explains how the system complies with Turkey's
**KVKK** (Kişisel Verilerin Korunması Kanunu, Law No. 6698) and the hackathon's
privacy & ethics requirements.

## 1. Purpose limitation — what we detect

LumiCity detects **urban assets only**:

- ✅ Streetlights / luminaires, utility & lighting poles
- ✅ Road segment geometry, road type, illumination density

It explicitly does **not** perform, and contains no code for:

- ❌ Face recognition or identity detection
- ❌ Person profiling, re-identification, or tracking
- ❌ Vehicle tracking or license-plate recognition / reading

The computer-vision model used (`facebook/mask2former-swin-large-mapillary-vistas-panoptic`)
is queried **only** for the Mapillary-Vistas classes corresponding to lighting
infrastructure. People and vehicles are treated purely as regions to anonymize,
never as data to extract.

## 2. Anonymization — irreversible, before storage

Every frame is anonymized **before** detection, storage, or display, in this order:

1. Detect face regions and license-plate regions.
2. Apply an **irreversible Gaussian blur** (destructive pixel operation — not a
   reversible mask or mosaic) to those regions.
3. Only the anonymized frame continues through the pipeline.

Implementation: [`ai/pipeline/anonymize.py`](../ai/pipeline/anonymize.py),
invoked by both the offline pipeline ([`ai/pipeline/run.py`](../ai/pipeline/run.py))
and the live worker ([`ai/worker/main.py`](../ai/worker/main.py)). Each stored
`lighting_analyses` record carries `anonymized`, `faces_blurred`, and
`plates_blurred` counters so anonymization is auditable end-to-end and surfaced
in the UI.

## 3. Data minimization & storage

- **No raw imagery is stored.** The pipeline persists only derived metadata
  (fixture counts, density, risk score, coordinates) and, optionally, the
  *already-anonymized* frame for demo display.
- **No raw data in the repository.** Source images, model weights, datasets, and
  `.env` secrets are excluded via [`.gitignore`](../.gitignore).
- **No unencrypted cloud storage** of raw data. Managed Postgres on Render holds
  only anonymized, derived records.
- **Seed data** ([`data/seed/segments.json`](../data/seed/segments.json)) is
  fully derived/aggregated — it contains no personal data.

## 4. Lawful basis & transparency

- Processing serves a **public-interest task** (municipal safety & energy
  efficiency) and uses imagery already published for that geography.
- The system is transparent: the dashboard and mobile app display, per analysis,
  exactly what was detected and how many faces/plates were blurred.

## 5. Retention & deletion

- Raw imagery is **transient**: fetched, anonymized in memory, and discarded.
  It is never written to durable storage in raw form.
- Any cached anonymized frames and all derived records are deletable on request.
- **Post-hackathon:** delete all raw/intermediate images and any API caches.
  Run: `make data-purge` (see below) or manually remove `ai/.cache/`, `ai/data/`,
  and revoke the Google Street View / HuggingFace API keys.

```bash
# Purge any local raw/intermediate imagery and caches
rm -rf ai/.cache ai/data/raw ai/data/tmp
# Revoke API keys in the Google Cloud console & HuggingFace settings
```

## 6. Responsibilities summary

| Requirement | Status | Where |
|-------------|--------|-------|
| Detect urban objects only | ✅ | `ai/pipeline/detect.py` |
| No face/identity/plate recognition | ✅ | no such code paths exist |
| Irreversible face/plate blur | ✅ | `ai/pipeline/anonymize.py` |
| No raw data in public repo | ✅ | `.gitignore` |
| No unencrypted raw cloud storage | ✅ | only derived records persisted |
| Delete raw images post-event | ✅ | §5 deletion procedure |
