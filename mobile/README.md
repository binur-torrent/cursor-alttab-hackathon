# LumiCity AI — Mobile (Expo)

Minimal field companion for municipal crews and citizens. Reads the same
`masterfabric-go` lighting API as the web dashboard.

## Features

- **Nearby risks** — map + list of the highest-risk street segments (markers
  colored by lighting-risk). Tap a segment for its detail + KVKK summary.
- **Report a dark spot** — uses your GPS location to run on-demand lighting
  analysis (`POST /api/v1/lighting/analyze`). Faces/plates are anonymized
  upstream by the AI worker.

## Run

```bash
npm install
# point the app at the backend (localhost won't resolve on a device):
echo "EXPO_PUBLIC_API_URL=http://<your-lan-ip>:8081" > .env
npx expo start
```

Open in Expo Go (iOS/Android) or a simulator. Uses `react-native-maps` and
`expo-location`.

## Config

| Env | Description |
| --- | --- |
| `EXPO_PUBLIC_API_URL` | Backend base URL (Render deployment or LAN IP for dev) |
