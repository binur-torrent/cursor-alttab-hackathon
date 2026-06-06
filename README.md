# LumiCity AI

**Intelligent urban lighting analysis and simulation for Istanbul.**

LumiCity AI combines HuggingFace computer-vision models, geolocated street-level
imagery, and an interactive digital twin to give municipalities a decision-support
tool for adaptive streetlight network design. It detects streetlight fixtures from
imagery, scores each street segment for lighting adequacy / safety risk, and lets
planners simulate energy-vs-safety trade-offs on a live map.

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
