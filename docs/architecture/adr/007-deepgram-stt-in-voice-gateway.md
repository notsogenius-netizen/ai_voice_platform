# ADR 007 — Deepgram STT in Voice Gateway (Phase 2)

**Status:** Accepted  
**Date:** 2026-08-27

## Decision

1. Implement **streaming speech-to-text in Phase 2 inside `voice-gateway`**, not in `ai-orchestrator`.
2. Use **Deepgram** as the single Phase 2 STT provider over its listen WebSocket API.
3. Define a small **`internal/stt` interface** (`Client`, `Stream`, `Session`, `Transcript`) with one implementation (`internal/stt/deepgram`).
4. Decode browser Opus → PCM in the gateway using **pure Go** (`thesyncim/gopus` + local resampling) — no system `libopus` / CGO requirement for local dev.

## Alternatives

1. STT in `ai-orchestrator` from day one (premature — orchestrator has no conversation loop yet)
2. Multi-provider STT matrix (Whisper, AssemblyAI, etc.) in Phase 2
3. LiveKit SDK `PCMRemoteTrack` + system libopus (rejected for local dev friction — requires `pkg-config` / libopus)
4. Send raw Opus to Deepgram (rejected — pipeline standardized on linear16 @ 16 kHz for provider clarity)

## Rationale

- Phase 2 goal is **media → text** only; the gateway already subscribes to browser audio.
- Keeping STT at the media boundary minimizes hops and matches the planned realtime path.
- A thin interface preserves testability without building a provider framework.
- Pure Go Opus decode keeps `make run-voice-gateway` working on macOS without Homebrew codec deps.

## Tradeoffs

| Upside | Downside |
|--------|----------|
| End-to-end STT with existing Phase 1 stack | STT logic temporarily colocated with media code |
| One env var (`DEEPGRAM_API_KEY`) enables STT | Deepgram account/key required for transcript verification |
| Session correlation (room, participant, track) in logs | Transcripts not persisted yet (Phase 6+) |

## Consequences

- `voice-gateway` opens a Deepgram stream per subscribed browser audio track when STT is configured.
- Partial and final transcripts are logged with `room`, `identity`, and `track` IDs.
- STT failure degrades gracefully: PCM path and verification tone continue; errors are logged.
- Phase 3 will introduce `ai-orchestrator`; transcript handoff to the orchestrator is a later wiring step.
