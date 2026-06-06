# Smart Lamp Prototype

A two-phone hackathon demo on top of our existing stack (`masterfabric-go` +
Next.js). One phone reads the ambient light from its camera; the other phone
turns into an adaptive street lamp that switches on in the dark — live over
WebSocket.

```
if luminance < 20  → lamp ON,  screen brightness = max (fullscreen white)
if luminance >= 20 → lamp OFF, screen brightness = min (near-black)
```

---

## Architecture

```
 Phone A (Sensor)                masterfabric-go                 Phone B (Lamp)
 ┌───────────────┐    wss/ws    ┌──────────────────┐   wss/ws   ┌───────────────┐
 │ camera frame  │ ───────────▶ │  /ws/lamp relay  │ ─────────▶ │ fullscreen    │
 │ → luminance   │   500ms      │  (room-based     │  fan-out   │ white / black │
 │   0..100      │   JSON       │   broadcast hub) │            │ ON / OFF      │
 └───────────────┘              └──────────────────┘            └───────────────┘
```

- **Backend (`masterfabric-go`)** hosts a dumb, room-based WebSocket relay. It
  keeps no state and persists nothing — it just fans out each JSON frame to the
  other clients in the same `room`. Lives in
  `internal/infrastructure/http/handler/lamp/` (infra-only; no domain/DB).
- **Two front-ends, same protocol:**
  1. **Go-served pages** at `/lamp/sensor` and `/lamp/screen` (self-contained
     HTML, embedded via `//go:embed`). **Same origin as the relay** → a single
     HTTPS tunnel gives the camera (`getUserMedia` needs a secure context) and
     `wss://` for free. This is the bulletproof live path.
  2. **Next.js pages** at `/lamp`, `/lamp/sensor`, `/lamp/screen` in `web/`
     (stack-native, nicer UI). Connects to a configurable WS URL.

### Why web instead of Expo?

For a *live* two-phone demo, opening a URL beats installing a native build:
no provisioning, no store, instant on any phone. The luminance + WebSocket
logic is identical in Expo — swap `getUserMedia`/`<video>` for
[`expo-camera`](https://docs.expo.dev/versions/v56.0.0/sdk/camera/) and read
pixels via a frame processor; the relay and message shape don't change. The
`mobile/` Expo app stays focused on the field-survey tool.

---

## WebSocket implementation

Relay endpoint: `GET /ws/lamp?room=<id>&role=<sensor|screen>`

- `room` pairs the two phones (default `demo`).
- `role` is informational. The relay forwards every text frame to all *other*
  clients in the room (slow clients are dropped, not blocked — only the latest
  luminance matters).
- Keep-alive via ping/pong; clients auto-reconnect.

Message shape (sensor → screen), sent every 500 ms:

```json
{ "luminance": 12, "lamp": "ON", "threshold": 20, "ts": 1780000000000 }
```

Key files:
- `internal/infrastructure/http/handler/lamp/hub.go` — rooms + fan-out.
- `internal/infrastructure/http/handler/lamp/handler.go` — upgrade, read/write
  pumps, static pages.
- `internal/shared/middleware/logging.go` — added an `http.Hijacker`
  passthrough so the WS upgrade survives the global logging middleware.

## Camera luminance estimation

Each frame is drawn to a tiny 64×48 canvas; we average the Rec. 601 relative
luminance of every pixel and map `0..255 → 0..100`:

```
luma  = 0.299*R + 0.587*G + 0.114*B      (per pixel)
score = round(avg(luma) / 255 * 100)     (0..100)
```

Downscaling makes it fast and naturally denoises. Threshold for ON/OFF is `20`.

## UI

- **Phone A (sensor):** live camera preview, big 0–100 luminance gauge with bar,
  computed lamp state, last-update timestamp, WS connection dot.
- **Phone B (lamp):** fullscreen. **White** at max when ON, **near-black** when
  OFF; overlays the current luminance, `ON/OFF` state, and last-update time.
  Goes stale if the sensor falls silent for >3 s.

---

## Run it live (two phones)

The robust single-origin path — phones load the page **and** the socket from the
same HTTPS host, so the camera and `wss://` just work.

```bash
# 1. Start the backend (DB optional — the relay needs neither Postgres nor Redis)
cd backend
SERVER_PORT=8081 go run ./cmd/server

# 2. Expose it over HTTPS (camera requires a secure context on phones)
ngrok http 8081           # or: cloudflared tunnel --url http://localhost:8081
```

Open `https://<tunnel-host>/lamp` on a laptop → scan the two QR codes with the
two phones (Phone A = Sensor, Phone B = Lamp), tap **Start** on each. Cover
Phone A's camera → Phone B lights up white. Same `?room=` pairs them; use
distinct rooms to run several pairs at once.

### Next.js version (same WiFi / polished UI)

```bash
cd web
# point the WS at an HTTPS/WSS tunnel of the backend for phone cameras:
NEXT_PUBLIC_WS_URL=wss://<backend-tunnel-host>/ws/lamp npm run dev
```

Then open `/lamp` on the web app. On `localhost` it works without HTTPS; on real
phones both the page and the socket must be HTTPS/WSS.
