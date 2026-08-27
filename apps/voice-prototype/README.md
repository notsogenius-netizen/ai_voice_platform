# Voice Prototype (Phase 1–3)

Minimal browser client: request a session from `voice-gateway`, join LiveKit, publish microphone audio, and play the gateway verification tone.

When `DEEPGRAM_API_KEY` and `AI_ORCHESTRATOR_URL` are configured (with orchestrator running and `LLM_API_KEY` set), speaking into the mic produces STT logs on the gateway and LLM replies on `ai-orchestrator` (spoken replies are Phase 4 — see [tts-phase4.md](../../docs/architecture/tts-phase4.md)).

See [docs/architecture/stt-phase2.md](../../docs/architecture/stt-phase2.md), [docs/architecture/orchestrator-phase3.md](../../docs/architecture/orchestrator-phase3.md), and [docs/architecture/tts-phase4.md](../../docs/architecture/tts-phase4.md).

## Prerequisites

1. LiveKit: `make livekit-up`
2. ai-orchestrator: `make run-ai-orchestrator` (optional for Phase 1–2 only)
3. voice-gateway: `make run-voice-gateway`
4. CORS origin must match Vite (`VOICE_GATEWAY_CORS_ORIGIN=http://127.0.0.1:5173`)

## Run

```bash
make run-voice-prototype
# or:
cd apps/voice-prototype && npm install && npm run dev
```

Open http://127.0.0.1:5173 — click **Connect + publish mic**, allow microphone access, and listen for a short verification tone from `voice-gateway`.

Optional: override gateway URL with `VITE_VOICE_GATEWAY_URL=http://127.0.0.1:8080`.
