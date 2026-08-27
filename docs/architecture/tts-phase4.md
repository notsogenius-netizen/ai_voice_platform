# Phase 4 — Text-to-Speech (Audible Replies)

Close the voice loop: assistant LLM text becomes spoken audio the caller hears in the browser.

```
Browser → LiveKit → voice-gateway → Deepgram (STT)
                         │
                         └── final transcript (HTTP)
                                    ▼
                            ai-orchestrator → LLM
                                    │
                                    └── reply stream (SSE text)
                                              ▼
                         voice-gateway → Deepgram Speak (Ogg Opus) → LiveKit → Browser
```

Phase 3 verified **speak → STT → LLM** via logs. Phase 4 adds **→ speech**.

Decision record: [ADR 009](adr/009-deepgram-tts-in-voice-gateway.md).

## Ownership

| Concern | Owner |
|---------|--------|
| Conversation + LLM (SSE text) | `ai-orchestrator` (unchanged) |
| TTS synthesis + LiveKit publish | `voice-gateway` |

TTS appears under the AI capability stack in the architecture overview; **runtime ownership** is the gateway media plane (same rationale as STT in ADR 007).

## Prerequisites

1. Phase 1–3 stack working ([livekit-local.md](livekit-local.md), [stt-phase2.md](stt-phase2.md), [orchestrator-phase3.md](orchestrator-phase3.md)).
2. Repo-root `.env` copied from [`.env.example`](../../.env.example).
3. **`DEEPGRAM_API_KEY`** set (shared with Phase 2 STT; TTS is disabled when unset).
4. For end-to-end turns: **`LLM_API_KEY`** and **`AI_ORCHESTRATOR_URL`**.

## Environment

| Variable | Default | Purpose |
|----------|---------|---------|
| `DEEPGRAM_API_KEY` | unset | Enables Deepgram STT and TTS when set |
| `DEEPGRAM_SPEAK_URL` | `https://api.deepgram.com/v1/speak` | Override Speak REST endpoint |
| `TTS_MODEL` | `aura-2-thalia-en` | Deepgram Aura voice model |

Example:

```env
DEEPGRAM_API_KEY=<deepgram-api-key>
# DEEPGRAM_SPEAK_URL=https://api.deepgram.com/v1/speak
# TTS_MODEL=aura-2-thalia-en
AI_ORCHESTRATOR_URL=http://127.0.0.1:8081
LLM_API_KEY=<llm-api-key>
```

## Feature slices

| Slice | Deliverable | Status |
|-------|-------------|--------|
| **4.0 Docs** | This guide + [ADR 009](adr/009-deepgram-tts-in-voice-gateway.md) | Done |
| **4.1 TTS client** | `internal/tts` + Deepgram Speak (Ogg Opus); env gate | Done |
| **4.2 Publish reply audio** | `NewLocalReaderTrack` publish into the LiveKit room | Done |
| **4.3 End-to-end turn** | Final STT → orchestrator → TTS → publish | Done |
| **4.4 Streaming TTS** | Sentence-level speak as SSE chunks arrive | Done |
| **4.5 Barge-in** | Stop playback when caller speaks again | Done |

## Run locally

Four terminals from the repository root:

```bash
make livekit-up
make run-ai-orchestrator
make run-voice-gateway
make run-voice-prototype
```

Open http://127.0.0.1:5173 → **Connect + publish mic** → speak, pause, and listen for the spoken reply.

### TTS client unit tests (no browser)

```bash
cd services/voice-gateway && go test ./internal/tts/... ./internal/roombot/ ./internal/config/
```

## Verification checklist

Mark Phase 4 complete only after all items pass.

### Startup

- [ ] Gateway logs `tts: enabled provider=deepgram model=... speak_url=...` when `DEEPGRAM_API_KEY` is set
- [ ] Gateway logs `tts: disabled ...` when the key is unset (Phases 1–3 still work)
- [ ] `go test ./...` passes in `services/voice-gateway`

### Happy path

- [ ] Speak → gateway `stt final` → orchestrator LLM reply → gateway TTS → spoken reply in browser
- [ ] Gateway logs `tts: synthesized room=sess_... bytes=...` and `roombot: publishing reply audio room=sess_...`
- [ ] Longer replies begin speaking before the full LLM stream finishes (sentence flush)
- [ ] Speaking again during a reply logs `roombot: barge-in` and stops agent audio
- [ ] Multi-turn: second utterance still gets a spoken reply
- [ ] Session correlation: same `room=` across STT, orchestrator forward, and TTS publish logs

### Errors / degradation

- [ ] Invalid Deepgram key: gateway stays up; TTS errors logged; STT/LLM path as configured
- [ ] Orchestrator down: STT continues; no reply audio for that turn
- [ ] `DEEPGRAM_API_KEY` unset: no TTS; Phase 3 log-only LLM behavior when orchestrator is enabled

## Expected log snippets

```
tts: enabled provider=deepgram model=aura-2-thalia-en speak_url=https://api.deepgram.com/v1/speak
stt final room=sess_... text="..."
orchestrator: forwarded turn room=sess_...
tts: synthesized room=sess_... bytes=...
roombot: publishing reply audio room=sess_...
roombot: barge-in room=sess_... identity=... track=...
```

Orchestrator continues to own `llm: reply` text logs; gateway does not dump full reply text.

## Code map (`services/voice-gateway`)

| Path | Role |
|------|------|
| `internal/tts/tts.go` | TTS interface + request/audio types |
| `internal/tts/deepgram/` | Deepgram Speak REST client (Ogg Opus) |
| `internal/config/` | `TTS_MODEL` / speak URL loading |
| `internal/roombot/reply_audio.go` | Ogg Opus publish + barge-in |
| `internal/roombot/sentence.go` | SSE chunk → sentence flush |
| `internal/roombot/stt_pipe.go` | Final transcript → StreamTurn → TTS → publish |
| `internal/orchestrator/httpclient/` | SSE `StreamTurn` with chunk callbacks |

Existing anchors:

| Path | Role |
|------|------|
| `internal/roombot/tone.go` | Phase 1 verification tone publish (same Ogg/Opus publish family) |
| `internal/roombot/bot.go` | Room join; shared reply playback per session |

## Out of scope (Phase 4)

- Browser UI for transcripts or AI text (display remains optional / later)
- Forwarding partial STT to the LLM (Phase 5)
- Persisting calls/transcripts (Phase 6)
- Redis conversation state (Phase 7)
- Multi-tenant agent / voice configuration (Phase 8)
- RAG / tool calling (Phases 9–10)
- Multi-provider TTS
- Deepgram streaming Speak WebSocket (REST sentence clips are enough for Phase 4)
