# ADR 009 — Deepgram TTS in Voice Gateway (Phase 4)

**Status:** Accepted  
**Date:** 2026-08-27

## Decision

1. Implement **text-to-speech in Phase 4 inside `voice-gateway`**, not in `ai-orchestrator`.
2. Use **Deepgram** as the single Phase 4 TTS provider via its Speak REST API (`POST /v1/speak`).
3. Reuse **`DEEPGRAM_API_KEY`** (same credential as Phase 2 STT); optional overrides for speak URL and model.
4. Request **Ogg Opus** (`encoding=opus&container=ogg`) and publish with **`lksdk.NewLocalReaderTrack`** — same CGO-free publish family as the Phase 1 verification tone (avoid `PCMLocalTrack` / system libopus).
5. Define a small **`internal/tts` interface** (`Client`, synthesize request/response types) with one implementation (`internal/tts/deepgram`).
6. Keep **`ai-orchestrator` text-only** for Phase 4: LLM replies continue as SSE text; the gateway synthesizes speech and publishes audio to LiveKit.
7. Stream SSE chunks into **sentence-sized Speak calls** (time-to-first-audio), and **barge-in** by interrupting playback when STT emits new caller speech.

## Alternatives

1. TTS inside `ai-orchestrator` (rejected for Phase 4 — orchestrator has no LiveKit media path; would add audio hop back to gateway)
2. Browser Web Speech API / client-side TTS (rejected — not a real media-plane path; diverges from LiveKit publish model)
3. Multi-provider TTS matrix (ElevenLabs, OpenAI, etc.) in Phase 4
4. LiveKit `PCMLocalTrack` + linear16 Speak (rejected — pulls CGO/`pkg-config` via `media-sdk` / libopus; conflicts with Phase 2 pure-Go local-dev goal)
5. Separate TTS API key / service (deferred — one Deepgram key is enough for local Phase 4)
6. Deepgram streaming Speak WebSocket from day one (deferred — REST per sentence is enough)

## Rationale

- Architectural principle: Voice Gateway = media/AI boundary; AI Orchestrator = conversation logic.
- Phase 1 already publishes Ogg/Opus from the gateway; Phase 2 already owns Deepgram credentials.
- Architecture diagrams list TTS under the “AI stack” as a capability; **service ownership** follows the media plane, matching ADR 007 (STT in gateway).
- Sentence-level Speak from SSE chunks lowers time-to-first-audio without a streaming TTS WebSocket.
- A thin `internal/tts` interface mirrors `internal/stt` and preserves testability without a provider framework.

## Tradeoffs

| Upside | Downside |
|--------|----------|
| Clear media vs conversation split | Gateway grows (STT + TTS + LiveKit) |
| Same Deepgram key enables STT and TTS | Deepgram account required for audible verification |
| CGO-free local publish path | One Speak HTTP call per sentence (not sample-streaming) |
| Orchestrator stays text-only | Barge-in uses STT activity, not a dedicated VAD |

## Consequences

- Phase 4 docs: [tts-phase4.md](../tts-phase4.md).
- When TTS is configured and wired, the gateway synthesizes assistant reply text as Ogg Opus and publishes a spoken track into the session room.
- TTS failure degrades gracefully: STT and LLM turns continue; errors are logged; no audible reply for that turn.
- `StreamTurn` on the gateway orchestrator client exposes SSE chunk callbacks for sentence TTS.
- Partials → LLM (Phase 5), persistence (Phase 6), and Redis (Phase 7) stay out of Phase 4.
