# ADR 009 — Deepgram TTS in Voice Gateway (Phase 4)

**Status:** Accepted  
**Date:** 2026-08-27

## Decision

1. Implement **text-to-speech in Phase 4 inside `voice-gateway`**, not in `ai-orchestrator`.
2. Use **Deepgram** as the single Phase 4 TTS provider via its Speak REST API (`POST /v1/speak`).
3. Reuse **`DEEPGRAM_API_KEY`** (same credential as Phase 2 STT); optional overrides for speak URL, model, and sample rate.
4. Define a small **`internal/tts` interface** (`Client`, synthesize request/response types) with one implementation (`internal/tts/deepgram`).
5. Keep **`ai-orchestrator` text-only** for Phase 4: LLM replies continue as SSE text; the gateway synthesizes speech and publishes audio to LiveKit.
6. Deliver Phase 4 in **incremental slices** (client → publish → end-to-end turn → optional streaming TTS → barge-in). Early slices speak a **full assistant reply** after SSE `[DONE]`, not token-by-token audio.

## Alternatives

1. TTS inside `ai-orchestrator` (rejected for Phase 4 — orchestrator has no LiveKit media path; would add audio hop back to gateway)
2. Browser Web Speech API / client-side TTS (rejected — not a real media-plane path; diverges from LiveKit publish model)
3. Multi-provider TTS matrix (ElevenLabs, OpenAI, etc.) in Phase 4
4. Streaming TTS WebSocket from day one (deferred — slice 4.4 quality work after full-reply path works)
5. Separate TTS API key / service (deferred — one Deepgram key is enough for local Phase 4)

## Rationale

- Architectural principle: Voice Gateway = media/AI boundary; AI Orchestrator = conversation logic.
- Phase 1 already publishes audio from the gateway (verification tone); Phase 2 already owns Deepgram credentials and PCM conventions.
- Architecture diagrams list TTS under the “AI stack” as a capability; **service ownership** follows the media plane, matching ADR 007 (STT in gateway).
- Full-reply TTS first keeps the contract simple: accumulate SSE text → synthesize → publish.
- A thin `internal/tts` interface mirrors `internal/stt` and preserves testability without a provider framework.

## Tradeoffs

| Upside | Downside |
|--------|----------|
| Clear media vs conversation split | Gateway grows (STT + TTS + LiveKit) |
| Same Deepgram key enables STT and TTS | Deepgram account required for audible verification |
| Reuses Phase 1 publish path and Phase 3 SSE handoff | Full-reply TTS adds latency vs streaming TTS |
| Orchestrator unchanged for Phase 4 core | Barge-in and sentence-streaming deferred |

## Consequences

- Phase 4 docs: [tts-phase4.md](../tts-phase4.md).
- When TTS is configured and wired, the gateway synthesizes assistant reply text and publishes a spoken track into the session room so the browser hears it.
- TTS failure degrades gracefully: STT and LLM turns continue; errors are logged; no audible reply for that turn.
- Streaming sentence-level TTS and barge-in remain later Phase 4 slices, not prerequisites for “hear a reply.”
- Partials → LLM (Phase 5), persistence (Phase 6), and Redis (Phase 7) stay out of Phase 4.
