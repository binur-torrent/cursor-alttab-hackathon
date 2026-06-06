# LumiCity AI

**AI-powered urban lighting assessment & planning for municipalities.**

LumiCity AI combines HuggingFace computer-vision models, geolocated street-level
imagery, and an interactive map to help cities find where street lighting is
insufficient, understand *why*, and decide which fixes have the most impact.

A planner **clicks a street segment** on the map. LumiCity assesses the scene —
lamps, poles, trees/vegetation, buildings, road width, sidewalks, sky visibility —
and scores it on three transparent dimensions:

- **Lighting Sufficiency** (0–100) — lamp coverage vs. the recommended density
- **Occlusion** (0–100, higher = worse) — vegetation/built mass blocking light
- **Infrastructure Adequacy** (0–100) — poles, sidewalks, physical readiness

…rolled up into a single **Overall Lighting Score**. The platform then generates
ranked recommendations (install N lamp posts, increase brightness, trim
vegetation, schedule inspection) and lets the planner **adjust the segment** —
add lamps or trim trees and watch the score and the map recolor live, *before*
spending public money.

> Built for the Cursor Hackathon Istanbul. Urban Impact + Working AI + Strong Demo + Privacy Compliance.

---

## Monorepo layout

| Path        | Stack                | Purpose |
|-------------|----------------------|---------|
| `backend/`  | Go (masterfabric-go) | Mandatory clean/hexagonal backend + new `lighting` bounded context |
| `ai/`       | Python               | HuggingFace ingestion + detection pipeline, anonymizer, FastAPI live worker |
| `web/`      | Next.js              | Dashboard: Istanbul risk map, segment detail, simulation (Vercel) |
| `mobile/`   | Expo                 | Field companion app: nearby problem segments + capture/report |
| `data/seed/`| JSON                 | Anonymized, derived seed data produced by the AI pipeline |
| `docs/`     | Markdown             | KVKK compliance, architecture notes, demo script |
| `deployments/` | YAML              | Render blueprint |

## Architecture compliance

The backend **is** the [`masterfabric-go`](https://github.com/gurkanfikretgunak/masterfabric-go)
enterprise clean/hexagonal platform (Chi, pgx/PostgreSQL, Redis, goose, optional Kafka,
OpenTelemetry). We did **not** build a custom backend. LumiCity is implemented as a new
bounded context `lighting` that mirrors the existing layering exactly:

```
internal/domain/lighting        -> entities + repository interfaces (zero deps)
internal/application/lighting    -> use cases + DTOs
internal/infrastructure/postgres/lighting -> pgx repositories
internal/infrastructure/http/handler/lighting -> Chi handlers
internal/infrastructure/postgres/migrations  -> goose migrations
```

## Scoring model & key API

The scoring model is a single source of truth, implemented as **byte-for-byte
ports** in Go ([`scoring.go`](backend/internal/domain/lighting/model/scoring.go))
and Python ([`scoring.py`](ai/pipeline/scoring.py)) so precomputed seed scores and
live scores always agree (enforced by `.cursor/rules/scoring-consistency.mdc`).

Lighting endpoints (public, read by the dashboard):

| Method & path | Purpose |
|---------------|---------|
| `GET  /api/v1/lighting/segments/map` | All segments (with geometry) for the map |
| `GET  /api/v1/lighting/segments/{id}` | One segment's full detail |
| `POST /api/v1/lighting/segments/{id}/rescore` | **What-if**: apply an intervention (add lamps, trim vegetation, brightness); returns baseline vs projected scores + recommendations. `persist:true` writes it back so the map recolors |
| `GET  /api/v1/lighting/stats` | Network-wide KPIs |
| `POST /api/v1/lighting/analyze` | On-demand analysis of an arbitrary point |
| `POST /api/v1/lighting/simulate` | Network-wide adaptive-lighting scenario |

## Quick start

See per-component READMEs:
- Backend: [`backend/README.md`](backend/README.md) (upstream) and [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md)
- AI pipeline: [`ai/README.md`](ai/README.md)
- Web: [`web/README.md`](web/README.md)
- Mobile: [`mobile/README.md`](mobile/README.md)

```bash
# 1. Backend (needs Docker for Postgres)
cd backend && ./dev.sh        # or: make docker-up && make run

# 2. Web dashboard
cd web && npm install && npm run dev

# 3. AI pipeline (produces data/seed/segments.json)
cd ai && pip install -r requirements.txt && python -m pipeline.run --help
```

## Privacy / KVKK

LumiCity detects **urban assets only** (streetlights, poles). It performs **no** face
recognition, identity detection, profiling, or license-plate reading. Faces and plates
are irreversibly blurred before any storage or display. No raw imagery is committed to
this repo. See [`docs/KVKK_COMPLIANCE.md`](docs/KVKK_COMPLIANCE.md).

## AI / Cursor workflow

This project was built agentically in Cursor. See the
[AI & Cursor Workflow](docs/AI_WORKFLOW.md) doc for tools, prompting techniques, and
how AI accelerated each phase.
