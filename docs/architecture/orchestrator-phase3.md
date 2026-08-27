# Phase 3 — AI Orchestrator

Introduce `ai-orchestrator` as the conversation brain: final STT transcripts become streaming LLM replies.

```
Browser → LiveKit → voice-gateway → Deepgram (STT)
                         │
                         └── final transcript (HTTP)
                                    ▼
                            ai-orchestrator → LLM (OpenAI-compatible)
                                    │
                                    └── reply stream (SSE) → gateway logs handoff
```

TTS (spoken reply in the browser) is **Phase 4**. Phase 3 verifies **speak → STT → LLM** via logs.

Decision record: [ADR 008](adr/008-ai-orchestrator-phase3.md).

## Prerequisites

1. Phase 1–2 stack working ([livekit-local.md](livekit-local.md), [stt-phase2.md](stt-phase2.md)).
2. Repo-root `.env` copied from [`.env.example`](../../.env.example).
3. **`LLM_API_KEY`** set (Gemini or OpenAI-compatible provider).
4. Optional but recommended for end-to-end voice path: **`DEEPGRAM_API_KEY`** and **`AI_ORCHESTRATOR_URL`**.

## Environment

| Variable | Default | Purpose |
|----------|---------|---------|
| `AI_ORCHESTRATOR_ADDR` | `:8081` | Orchestrator HTTP listen address |
| `LLM_API_KEY` | unset | Enables LLM; leave empty to run health-only skeleton |
| `LLM_BASE_URL` | Gemini OpenAI-compat endpoint | OpenAI-compatible chat API base |
| `LLM_MODEL` | `gemini-3.6-flash` | Model name for the provider |
| `AGENT_SYSTEM_PROMPT` | built-in support agent prompt | Optional override |
| `AI_ORCHESTRATOR_URL` | unset | Gateway → orchestrator base URL (e.g. `http://127.0.0.1:8081`) |

Gemini (default):

```env
LLM_API_KEY=<gemini-api-key>
LLM_BASE_URL=https://generativelanguage.googleapis.com/v1beta/openai/
LLM_MODEL=gemini-3.6-flash
AI_ORCHESTRATOR_URL=http://127.0.0.1:8081
```

OpenAI:

```env
LLM_API_KEY=sk-...
LLM_BASE_URL=https://api.openai.com/v1
LLM_MODEL=gpt-4o-mini
```

## Run locally

Four terminals from the repository root:

```bash
make livekit-up
make run-ai-orchestrator
make run-voice-gateway
make run-voice-prototype
```

Open http://127.0.0.1:5173 → **Connect + publish mic** → speak clearly, then pause.

### Orchestrator-only smoke test (no browser)

```bash
make run-ai-orchestrator

curl -s -N -X POST http://127.0.0.1:8081/v1/sessions/my-room/turn \
  -H 'Content-Type: application/json' \
  -d '{"text":"What can you help me with?","is_final":true}'
```

Expected SSE:

```
data: {"text":"..."}

data: [DONE]
```

Partial transcripts return `202` with `{"status":"ignored","reason":"partial transcript"}`.

## API contract

`POST /v1/sessions/{id}/turn`

| Field | Type | Behavior |
|-------|------|----------|
| `text` | string | User utterance |
| `is_final` | bool | `false` → ignored; `true` → LLM turn |

Session ID is the LiveKit **room name** (same `sess_...` value from gateway STT logs).

## Log ownership

| Service | Logs |
|---------|------|
| **voice-gateway** | `stt partial/final`, `orchestrator: forwarded turn`, transport failures |
| **ai-orchestrator** | `turn: received`, `llm: reply` (full AI text), turn/LLM errors |

Gateway does **not** log full AI reply text; orchestrator owns conversation/LLM observability.

## Verification checklist

Mark Phase 3 complete only after all items pass.

### Startup

- [ ] Orchestrator logs `llm: enabled model=... base_url=...` when `LLM_API_KEY` is set
- [ ] Gateway logs `orchestrator: enabled url=http://127.0.0.1:8081` when `AI_ORCHESTRATOR_URL` is set
- [ ] `GET /healthz` on `:8081` returns `ok`
- [ ] `go test ./...` passes in `services/ai-orchestrator` and `services/voice-gateway`

### Happy path (keys set)

- [ ] Speak → gateway `stt final room=sess_... text="..."`
- [ ] Gateway `orchestrator: forwarded turn room=sess_...`
- [ ] Orchestrator `turn: received session=sess_... text="..."`
- [ ] Orchestrator `llm: reply session=sess_... text="..."`
- [ ] Same session ID keeps multi-turn context (second utterance references prior turn)

### Errors / degradation

- [ ] Orchestrator down: gateway logs `orchestrator: turn failed ...`; STT continues
- [ ] `LLM_API_KEY` unset: orchestrator runs; final turns return unavailable / llm not configured
- [ ] `AI_ORCHESTRATOR_URL` unset: gateway logs STT only (Phase 2 behavior)

## Code map

### `services/ai-orchestrator`

| Path | Role |
|------|------|
| `cmd/ai-orchestrator/` | Entrypoint, config, graceful shutdown |
| `internal/config/` | Env loading (`LLM_*`, listen addr) |
| `internal/llm/` | Streaming LLM interface |
| `internal/llm/openai/` | OpenAI-compatible SSE chat client |
| `internal/conversation/` | In-memory session history + turn handling |
| `internal/httpserver/` | `GET /healthz`, `POST /v1/sessions/{id}/turn` |

### `services/voice-gateway`

| Path | Role |
|------|------|
| `internal/orchestrator/` | Turn/Reply client interface |
| `internal/orchestrator/httpclient/` | HTTP client for turn API |
| `internal/roombot/stt_pipe.go` | Final transcript → orchestrator forward |

## Out of scope (Phase 3)

- TTS / audible AI replies (Phase 4)
- Browser UI for transcripts or AI text
- Persisting calls/transcripts (Phase 6)
- Redis conversation state (Phase 7)
- Multi-tenant agent configuration (Phase 8)
- RAG / tool calling (Phases 9–10)
