# ADR 006 — LiveKit Media Transport and Phase-1 Token Ownership

**Status:** Accepted  
**Date:** 2026-08-26

## Decision

1. Use **LiveKit** as the realtime WebRTC media transport for browser (and later SIP) voice sessions.
2. In Phase 1, **`voice-gateway` owns LiveKit access-token minting** and joins each session room as a server-side participant.

## Alternatives

1. Mint tokens in `api-gateway` from day one (control-plane edge)
2. Dedicated token service
3. Browser holds LiveKit API secrets (rejected — insecure)
4. Non-LiveKit WebRTC SFU / custom media stack

## Rationale

- LiveKit matches the browser-first media plane and keeps audio transport out of application code.
- Phase 1 needs one runnable backend; putting tokens on `voice-gateway` avoids standing up `api-gateway` early.
- The gateway already sits on the media/AI boundary, so joining rooms as a participant is a natural next step toward STT/TTS.
- Token ownership can move to the control plane later when auth/tenancy exists, without changing LiveKit itself.

## Tradeoffs

| Upside | Downside |
|--------|----------|
| Smallest Phase 1 surface area | Media service temporarily also issues join tokens |
| Clear path to subscribe/publish for AI pipeline | Control-plane concerns may later migrate off voice-gateway |
| Official Go SDK for server participants | Local Docker WebRTC tuning needed on macOS/Desktop |

## Consequences

- Browser clients obtain `{ room, livekit_url, token }` from `POST /v1/sessions`.
- `voice-gateway` never exposes `LIVEKIT_API_SECRET` to the browser.
- Phase 1 verification publishes a short tone from the gateway; STT added in Phase 2 (ADR 007).
