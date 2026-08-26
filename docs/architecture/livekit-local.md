# Local LiveKit (Phase 1)

How to run LiveKit for local browser voice development.

## Concepts (minimal)

| Concept | Meaning |
|---------|---------|
| **Room** | Named realtime session (participants join the same room) |
| **Participant** | A client in a room (browser user or server-side bot) |
| **Track** | A media stream (e.g. microphone audio) |
| **Access token** | JWT signed with API key/secret; grants identity, room join, and publish/subscribe grants |

The browser never holds the API secret. A backend mints short-lived tokens (Phase 1: `voice-gateway`).

## Start / stop

From the repository root:

```bash
make livekit-up    # docker compose up -d
make livekit-down  # docker compose down
```

Compose file: [`deployments/docker/docker-compose.livekit.yml`](../../deployments/docker/docker-compose.livekit.yml)  
Config: [`deployments/docker/livekit.yaml`](../../deployments/docker/livekit.yaml)

## Connection details (dev)

| Setting | Value |
|---------|--------|
| WebSocket URL | `ws://127.0.0.1:7880` |
| API key | `devkey` |
| API secret | `devsecret_ai_voice_platform_local_only` |

These match [`.env.example`](../../.env.example). **Dev-only** — do not reuse in production.

## Ports

| Port | Role |
|------|------|
| `7880` TCP | Signaling (WebSocket) |
| `7881` TCP | RTC over TCP fallback |
| `7882` UDP | RTC media (UDP mux; friendlier for Docker Desktop on macOS) |

## How the browser connects

1. Start LiveKit and `voice-gateway` (`make livekit-up`, `make run-voice-gateway`).
2. Start the prototype UI: `make run-voice-prototype` → http://127.0.0.1:5173
3. The page calls `POST /v1/sessions`, connects to LiveKit with the returned token, and publishes the microphone.

See [`apps/voice-prototype/README.md`](../../apps/voice-prototype/README.md).

## Health check

After `make livekit-up`:

```bash
curl -sS http://127.0.0.1:7880/
```

LiveKit responds on the HTTP signal port when the process is up. Check container status with:

```bash
docker compose -f deployments/docker/docker-compose.livekit.yml ps
```

## Phase 2 STT

With `DEEPGRAM_API_KEY` in `.env`, `voice-gateway` streams transcripts while you speak. See [stt-phase2.md](stt-phase2.md) for the full verification checklist.
