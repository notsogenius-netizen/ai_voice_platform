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

1. Backend creates a room name and mints an access token (identity + room + grants).
2. Browser connects to `LIVEKIT_URL` with that token via `livekit-client`.
3. Browser publishes microphone audio; other participants (including a server bot) can subscribe.

## Health check

After `make livekit-up`:

```bash
curl -sS http://127.0.0.1:7880/
```

LiveKit responds on the HTTP signal port when the process is up. Check container status with:

```bash
docker compose -f deployments/docker/docker-compose.livekit.yml ps
```
