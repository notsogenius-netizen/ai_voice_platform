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
                         voice-gateway → Deepgram (TTS) → LiveKit → Browser
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
4. For end-to-end turns (slice 4.3+): **`LLM_API_KEY`** and **`AI_ORCHESTRATOR_URL`**.

## Environment

| Variable | Default | Purpose |
|----------|---------|---------|
| `DEEPGRAM_API_KEY` | unset | Enables Deepgram STT and TTS when set |
| `DEEPGRAM_SPEAK_URL` | `https://api.deepgram.com/v1/speak` | Override Speak REST endpoint |
| `TTS_MODEL` | `aura-2-thalia-en` | Deepgram Aura voice model |
| `TTS_SAMPLE_RATE` | `24000` | linear16 PCM sample rate from Speak |

Example:

```env
DEEPGRAM_API_KEY=<deepgram-api-key>
# DEEPGRAM_SPEAK_URL=https://api.deepgram.com/v1/speak
# TTS_MODEL=aura-2-thalia-en
# TTS_SAMPLE_RATE=24000
AI_ORCHESTRATOR_URL=http://127.0.0.1:8081
LLM_API_KEY=<llm-api-key>
```

## Feature slices

Implement and verify one slice at a time.

| Slice | Deliverable | Done when |
|-------|-------------|-----------|
| **4.0 Docs** | This guide + [ADR 009](adr/009-deepgram-tts-in-voice-gateway.md) | Docs merged; links from architecture index |
| **4.1 TTS client** | `internal/tts` + Deepgram Speak client; env gate | Smoke: text → audio bytes; disabled when key unset; `go test` passes |
| **4.2 Publish reply audio** | Synthesize and publish a track to the LiveKit room | Browser hears spoken audio for a test/full reply string |
| **4.3 End-to-end turn** | SSE accumulate → TTS → publish on each final STT turn | Real loop: speak → STT → LLM → hear reply |
| **4.4 Streaming TTS** | Sentence-level speak as SSE chunks arrive | Lower time-to-first-audio (optional quality) |
| **4.5 Barge-in** | Stop/cancel playback when caller speaks again | Interruptible agent audio |

## Run locally

Same four terminals as Phase 3 (from repository root):

```bash
make livekit-up
make run-ai-orchestrator
make run-voice-gateway
make run-voice-prototype
```

Open http://127.0.0.1:5173 → **Connect + publish mic** → speak, pause, and (from slice 4.3) listen for the spoken reply.

### TTS-only smoke (slice 4.1 — no browser)

After the Deepgram TTS client lands, unit/integration tests under `services/voice-gateway` synthesize a fixed string to PCM/WAV bytes without LiveKit. Exact command will be documented with slice 4.1.

## Verification checklist

Mark Phase 4 complete only after slices **4.1–4.3** pass. Slices **4.4–4.5** are optional quality follow-ups.

### Startup (from 4.1)

- [ ] Gateway logs `tts: enabled provider=deepgram model=... sample_rate=...` when `DEEPGRAM_API_KEY` is set
- [ ] Gateway logs `tts: disabled ...` when the key is unset (Phases 1–3 still work)
- [ ] `go test ./...` passes in `services/voice-gateway`

### Publish path (from 4.2)

- [ ] Gateway can publish synthesized audio into the session room
- [ ] Browser hears spoken audio (prototype already subscribes to remote tracks for the verification tone)
- [ ] TTS failure is logged; room/STT path stays up

### Happy path (from 4.3)

- [ ] Speak → gateway `stt final` → orchestrator LLM reply → gateway TTS → spoken reply in browser
- [ ] Session correlation: same `room=` across STT, orchestrator forward, and TTS publish logs
- [ ] Multi-turn: second utterance still gets a spoken reply

### Errors / degradation

- [ ] Invalid Deepgram key: gateway stays up; TTS errors logged; STT/LLM path as configured
- [ ] Orchestrator down: STT continues; no reply audio for that turn
- [ ] `DEEPGRAM_API_KEY` unset: no TTS; Phase 3 log-only LLM behavior when orchestrator is enabled

## Expected log snippets (target after 4.3)

```
tts: enabled provider=deepgram model=aura-2-thalia-en sample_rate=24000 speak_url=https://api.deepgram.com/v1/speak
stt final room=sess_... text="..."
orchestrator: forwarded turn room=sess_...
tts: synthesized room=sess_... bytes=...
roombot: publishing reply audio room=sess_...
```

Orchestrator continues to own `llm: reply` text logs; gateway does not need to dump full reply text.

## Planned code map (`services/voice-gateway`)

Paths land with implementation slices; names may adjust slightly.

| Path | Role | Slice |
|------|------|-------|
| `internal/tts/tts.go` | TTS interface + request/audio types | 4.1 |
| `internal/tts/deepgram/` | Deepgram Speak REST client | 4.1 |
| `internal/config/` | `TTS_*` / speak URL loading | 4.1 |
| `internal/roombot/` (reply publish) | PCM/file track publish for assistant audio | 4.2 |
| Orchestrator SSE consumer path | Accumulate reply → TTS → publish | 4.3 |

Existing anchors:

| Path | Role |
|------|------|
| `internal/roombot/tone.go` | Phase 1 verification tone publish (pattern to reuse) |
| `internal/orchestrator/httpclient/` | Phase 3 SSE turn client |
| `internal/roombot/stt_pipe.go` | Final transcript → orchestrator forward |

## Out of scope (Phase 4)

- Browser UI for transcripts or AI text (display remains optional / later)
- Forwarding partial STT to the LLM (Phase 5)
- Persisting calls/transcripts (Phase 6)
- Redis conversation state (Phase 7)
- Multi-tenant agent / voice configuration (Phase 8)
- RAG / tool calling (Phases 9–10)
- Multi-provider TTS
