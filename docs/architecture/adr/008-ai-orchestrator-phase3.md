# ADR 008 — AI Orchestrator and LLM Boundary (Phase 3)

**Status:** Accepted  
**Date:** 2026-08-27

## Decision

1. Introduce **`ai-orchestrator`** as a separate deployable service that owns conversation context, agent instructions, and LLM interaction.
2. Keep **STT in `voice-gateway`** (ADR 007); forward only **final** transcripts to the orchestrator over HTTP.
3. Use an **OpenAI-compatible** chat completions API (`/chat/completions` + SSE streaming) behind a small `internal/llm` interface, with **Gemini** as the default provider via Google’s compatibility endpoint.
4. Use the LiveKit **room name** as the conversation **session ID**.
5. Keep conversation history **in-memory** for Phase 3 (Redis / durability come later).

## Alternatives

1. Call the LLM from `voice-gateway` (rejected — mixes media plane with conversation brain)
2. Native Gemini API only, no OpenAI-compat layer (rejected — harder to swap providers for local/dev)
3. Require OpenAI specifically (rejected — Gemini free tier is sufficient for Phase 3)
4. gRPC between gateway and orchestrator (deferred — HTTP + SSE is enough for text turns)
5. Forward partial STT transcripts to the LLM (deferred — Phase 5 quality work)

## Rationale

- Architectural principle: Voice Gateway = media/AI boundary; AI Orchestrator = conversation logic.
- Final-only turns keep Phase 3 simple and avoid noisy LLM calls on partial STT.
- OpenAI-compatible HTTP lets Gemini, OpenAI, and other providers share one client.
- Room-as-session-ID reuses existing STT correlation without inventing a new ID scheme yet.
- In-memory state matches “smallest useful version” before Redis (Phase 7) and call persistence (Phase 6).

## Tradeoffs

| Upside | Downside |
|--------|----------|
| Clear service boundary for LLM logic | Extra hop (gateway → orchestrator) on each final turn |
| Provider-neutral env (`LLM_API_KEY`, `LLM_BASE_URL`, `LLM_MODEL`) | Gemini model names change; defaults may need updates |
| Streaming SSE ready for Phase 4 TTS handoff | In-memory history lost on orchestrator restart |
| Gateway degrades if orchestrator is down | No audible reply until Phase 4 |

## Consequences

- `POST /v1/sessions/{id}/turn` with `{ "text", "is_final" }` is the Phase 3 contract.
- Partials return `202 ignored`; finals stream SSE chunks then `data: [DONE]`.
- Gateway sets `AI_ORCHESTRATOR_URL` to enable forwarding; without it, STT-only Phase 2 behavior remains.
- Orchestrator logs own conversation turns and LLM replies; gateway logs STT and forward success/failure only.
- TTS, barge-in, and durable session state remain out of scope for Phase 3.
