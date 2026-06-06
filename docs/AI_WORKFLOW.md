# AI & Cursor Workflow

LumiCity AI was built **agentically in Cursor**. This documents how AI tooling
was used — a judged criterion ("AI Adaptation").

## Tools & models

- **Cursor IDE** with the agent (Claude Opus-class model, extended thinking) as
  the primary implementation driver.
- **Cursor rules** (`.cursor/rules/`) encode the project's hard constraints
  (masterfabric-go architecture, KVKK privacy) so every agent turn respects them.
- **HuggingFace** models/datasets power the actual product:
  - Detection: `facebook/mask2former-swin-large-mapillary-vistas-panoptic`
    (Mapillary-Vistas classes for streetlights/poles).
  - Datasets: `Reubencf/streetview-global`, `nyuuzyou/streetview` (filtered to
    Istanbul) for seeding the digital twin.

## How AI accelerated each phase

| Phase | How the agent helped |
|-------|----------------------|
| Architecture study | Read the `masterfabric-go` codebase, identified the domain/application/infrastructure layering, and a representative vertical slice (`tenant`) to mirror. |
| Backend slice | Generated the entire `lighting` bounded context (entities, repos, use cases, Chi handlers, goose migration) matching upstream conventions in one pass. |
| Consistency | Ported the scoring model 1:1 between Python (`scoring.py`) and Go (`scoring.go`) and wrote Go unit tests to lock the behavior. |
| Frontend | Scaffolded the Next.js + Leaflet dashboard and Expo app, wiring them to real endpoints. |
| Validation | Ran the backend against a real Postgres, auto-seeded 143 segments, and curl-tested every endpoint before committing. |

## Prompting techniques that worked

- **Constraint-first**: the mandatory stack (masterfabric-go, HF data, KVKK) was
  stated up front and encoded as always-apply Cursor rules, preventing drift.
- **Mirror an existing slice**: "implement `lighting` the way `tenant` is
  implemented" produced idiomatic, review-ready code.
- **Single source of truth for logic**: define scoring once, port deterministically,
  and test — so seed data and live analysis never disagree.
- **Verify, then commit**: each milestone was built, run, and exercised against a
  live DB/worker before a small, meaningful commit (continuous Git history).

## Reproducibility

- Deterministic seed generator (`ai/pipeline/generate_seed.py`) reproduces the
  digital twin with no heavy downloads.
- Shared scoring logic guarantees identical results across Python and Go.
- One-command local stack via `backend/dev.sh` + `npm run dev`.
