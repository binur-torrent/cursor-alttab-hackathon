# Architecture

LumiCity AI is a monorepo with a Go backend (the mandatory `masterfabric-go`
platform), a Python AI tier, a Next.js dashboard, and an Expo app.

## System diagram

```
                 ┌──────────────┐     ┌──────────────┐
   Planners ───▶ │  Next.js web │     │  Expo mobile │ ◀── Field crews
                 │  (Vercel)    │     │              │
                 └──────┬───────┘     └──────┬───────┘
                        │  HTTPS (REST)      │
                        ▼                    ▼
                 ┌────────────────────────────────────┐
                 │   masterfabric-go backend (Render)  │
                 │   internal/.../lighting (new ctx)   │
                 │   Chi · pgx · clean architecture    │
                 └───────┬───────────────────┬─────────┘
                         │                   │ POST /analyze/point
                         ▼                   ▼
                 ┌───────────────┐   ┌────────────────────┐
                 │  PostgreSQL   │   │  Python AI worker  │
                 │ (anonymized   │   │  (FastAPI, Render) │
                 │  derived data)│   │  Street View + CV  │
                 └───────────────┘   └─────────┬──────────┘
                                               │
                            ┌──────────────────┴───────────────┐
                            │ Mask2Former (Mapillary-Vistas)    │
                            │ + KVKK anonymizer (faces/plates)  │
                            └───────────────────────────────────┘

   Offline:  HuggingFace datasets ──▶ ai/pipeline/run.py ──▶ data/seed/segments.json
                                       (filter · blur · detect · score · aggregate)
```

## Backend: lighting bounded context

We did **not** build a custom backend. LumiCity is a new bounded context inside
`masterfabric-go`, mirroring its existing layering exactly:

| Layer | Path | Contents |
|-------|------|----------|
| Domain | `internal/domain/lighting/` | `StreetSegment`, `LightFixture`, `LightingAnalysis`, scoring & simulation models, repository interfaces |
| Application | `internal/application/lighting/` | use cases (List/Get/Stats/Simulate/AnalyzeLive/IngestSeed) + DTOs + embedded seed |
| Infrastructure | `internal/infrastructure/postgres/lighting/` | pgx repositories + idempotent `EnsureSchema` |
| Transport | `internal/infrastructure/http/handler/lighting/` | Chi handlers; routes registered in `router.go` |
| AI adapter | `internal/infrastructure/aiworker/` | HTTP client implementing `usecase.AnalyzerPort` |

### Public API

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/v1/lighting/stats` | network KPIs + per-district breakdown |
| GET | `/api/v1/lighting/segments` | paginated, filtered list (risk desc) |
| GET | `/api/v1/lighting/segments/map` | all matching segments w/ geometry |
| GET | `/api/v1/lighting/segments/{id}` | detail (fixtures + anonymized frames); id = UUID or external id |
| POST | `/api/v1/lighting/simulate` | energy-vs-safety scenario |
| POST | `/api/v1/lighting/analyze` | on-demand point analysis (worker or fallback) |

## Key design decisions

- **One scoring model, two languages.** `ai/pipeline/scoring.py` and
  `internal/domain/lighting/model/scoring.go` are deterministic ports of each
  other, so precomputed seeds and live analysis always agree (Go unit-tested).
- **Zero-touch demo.** Anonymized seed (143 Istanbul segments) is embedded and
  auto-loaded on first boot (`LIGHTING_AUTOSEED`), and the schema is ensured at
  startup (`LIGHTING_ENSURE_SCHEMA`) — no manual migration step on Render.
- **Graceful AI degradation.** If `AI_WORKER_URL` is unset or the worker is down,
  `AnalyzeLive` falls back to a deterministic Go heuristic so the live demo never
  breaks.
