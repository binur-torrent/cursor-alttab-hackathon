# Demo Script (~4 minutes)

A tight, judge-ready walkthrough. Hits every scoring criterion: technical
functionality, accuracy, public benefit, AI adaptation, privacy.

## 0. Setup (before you present)

```bash
# Postgres
docker run -d --name lumicity-pg -e POSTGRES_USER=masterfabric \
  -e POSTGRES_PASSWORD=masterfabric -e POSTGRES_DB=masterfabric \
  -p 5433:5432 postgres:16-alpine

# Backend (auto-creates schema + seeds 143 segments)
cd backend && DB_HOST=localhost DB_PORT=5433 DB_USER=masterfabric \
  DB_PASSWORD=masterfabric DB_NAME=masterfabric DB_SSLMODE=disable \
  KAFKA_ENABLED=false SERVER_PORT=8081 go run ./cmd/server

# (optional) live AI worker
cd ai && python -m venv .venv && . .venv/bin/activate \
  && pip install -r worker/requirements.txt \
  && MODEL_BACKEND=heuristic uvicorn worker.main:app --port 8090
# then restart backend with AI_WORKER_URL=http://127.0.0.1:8090

# Web
cd web && npm install && npm run dev   # http://localhost:3000
```

## 1. The problem (20s)

> "Istanbul's streetlights are static. Some streets waste energy lit all night;
> others are dangerously dark. Municipalities have no data-driven way to
> prioritize. LumiCity turns street imagery into a lighting decision tool."

## 2. The map — public benefit + accuracy (60s)

- Open the dashboard. Point at the **KPI bar**: 143 segments, avg risk,
  high-risk count, risk distribution.
- The **map** is colored by lighting risk (green→red). "Each line is a real
  street segment scored from detected streetlight density vs. what its road type
  needs."
- Filter to **Critical** → the dark, dangerous corridors light up red.
- Click a red segment → **detail panel**: streetlight count, adequacy %,
  analyzed frames. Note the **"faces/plates blurred"** line — privacy is visible.

## 3. Simulation — the decision tool (60s)

- Switch to the **Simulate** tab. "What if we bring every critical & high-risk
  street to 90% adequacy, and dim well-lit streets 60% after midnight?"
- Drag the sliders → live KPIs: **risk down ~60%**, fixtures added, and the
  **energy / cost / CO₂** trade-off. "This is the energy-vs-safety conversation a
  city actually needs to have, quantified."
- Scope it to a single district to show targeted budgeting.

## 4. Analyze any address — live AI (45s)

- Switch to **Analyze**. Type a real address (e.g. *Bağdat Caddesi, Kadıköy*).
- It geocodes, calls the backend → AI worker → fetches Street View, **blurs faces
  & plates**, runs the CV model, scores it. Show the anonymized image + risk.
- "Same pipeline, on demand, for anywhere in the city."

## 5. Architecture & AI (40s)

- "Backend is the mandatory **masterfabric-go** clean-architecture platform — we
  added a `lighting` bounded context, not a custom backend."
- "Detection is a **HuggingFace** Mask2Former model; the digital twin is seeded
  from HuggingFace street-view datasets."
- "Built agentically in **Cursor** with rules encoding our architecture + KVKK
  constraints." (point to `.cursor/rules/`)

## 6. Privacy close (15s)

> "Every face and plate is irreversibly blurred before anything is stored. We
> detect lampposts, not people. No raw imagery is kept. Full KVKK write-up in
> `docs/KVKK_COMPLIANCE.md`."

## Fallback notes

- No internet / worker down? `Analyze` still returns a deterministic result
  (Go heuristic) — the demo never hard-fails.
- Mobile: `mobile/` Expo app shows nearby risks + GPS "report a dark spot".
