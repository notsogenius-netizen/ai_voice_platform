# Voice Prototype (Phase 1)

Minimal browser client: request a session from `voice-gateway`, join LiveKit, publish microphone audio, and play the gateway verification tone.

## Prerequisites

1. LiveKit: `make livekit-up`
2. voice-gateway: `make run-voice-gateway`
3. CORS origin must match Vite (`VOICE_GATEWAY_CORS_ORIGIN=http://127.0.0.1:5173`)

## Run

```bash
make run-voice-prototype
# or:
cd apps/voice-prototype && npm install && npm run dev
```

Open http://127.0.0.1:5173 — click **Connect + publish mic**, allow microphone access, and listen for a short verification tone from `voice-gateway`.

Optional: override gateway URL with `VITE_VOICE_GATEWAY_URL=http://127.0.0.1:8080`.
