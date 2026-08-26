# Phase 2 — Streaming Speech-to-Text

Convert live browser speech to streaming partial and final transcripts via Deepgram.

```
Browser → LiveKit → voice-gateway → Deepgram
                         │                │
                         └── PCM decode ──┘
                                    partial / final transcripts (logs)
```

Decision record: [ADR 007](adr/007-deepgram-stt-in-voice-gateway.md).

## Prerequisites

1. Phase 1 stack working ([livekit-local.md](livekit-local.md)).
2. Repo-root `.env` copied from [`.env.example`](../../.env.example).
3. **`DEEPGRAM_API_KEY`** set to a valid Deepgram API key (STT is disabled when unset).

Optional:

| Variable | Default | Purpose |
|----------|---------|---------|
| `DEEPGRAM_LISTEN_URL` | `wss://api.deepgram.com/v1/listen` | Override listen endpoint |
| `STT_SAMPLE_RATE` | `16000` | PCM sample rate sent to Deepgram |

## Run locally

Three terminals from the repository root:

```bash
make livekit-up
make run-voice-gateway
make run-voice-prototype
```

Open http://127.0.0.1:5173 → **Connect + publish mic** → speak clearly for a few seconds.

## Verification checklist

Mark Phase 2 complete only after all items pass.

### Startup

- [ ] Gateway logs `stt: enabled provider=deepgram sample_rate=16000 ...` when `DEEPGRAM_API_KEY` is set
- [ ] Gateway logs `stt: disabled ...` when the key is unset (Phase 1 still works)
- [ ] `go test ./...` passes in `services/voice-gateway`

### Happy path (key set)

- [ ] Browser connects; verification tone still plays
- [ ] Gateway logs `roombot: stt stream opened room=... identity=... track=...`
- [ ] Gateway logs `pcm: started ... sample_rate=16000`
- [ ] While speaking, gateway logs `pcm: streaming ... bytes=N` (every ~3s)
- [ ] Gateway logs `stt partial room=... text="..."` updating as you talk
- [ ] Gateway logs `stt final room=... text="..."` when you pause or finish a phrase

### Session correlation

- [ ] `room=` matches the session room name from `POST /v1/sessions`
- [ ] `identity=` matches the browser participant identity
- [ ] `track=` matches the subscribed audio track SID

### Error handling

- [ ] Invalid `DEEPGRAM_API_KEY`: gateway stays up; `roombot: stt open ...` error; tone still works
- [ ] Disconnect browser: no panic; `pcm: stopped` / track close logs
- [ ] Ctrl+C gateway: clean shutdown

## Expected log snippets

```
stt: enabled provider=deepgram sample_rate=16000 listen_url=wss://api.deepgram.com/v1/listen
voice-gateway listening on :8080
roombot: joined room=sess_... as voice-gateway
roombot: stt stream opened room=sess_... identity=browser-... track=TR_...
pcm: started room=sess_... identity=browser-... track=TR_... sample_rate=16000
pcm: streaming room=sess_... bytes=12345
stt partial room=sess_... identity=browser-... track=TR_... text="hello"
stt final room=sess_... identity=browser-... track=TR_... text="hello world"
```

## Code map

| Path | Role |
|------|------|
| `internal/stt/stt.go` | STT interface + session/transcript types |
| `internal/stt/deepgram/` | Deepgram WebSocket client |
| `internal/audio/pcm/` | Opus → PCM decode + resample |
| `internal/roombot/track.go` | Per-track PCM + STT lifecycle |
| `internal/roombot/stt_pipe.go` | PCM → Deepgram write + transcript read |

## Out of scope (Phase 2)

- Transcripts in the browser UI
- Persisting transcripts (`call-service` — Phase 6)
- Sending transcripts to `ai-orchestrator` (Phase 3)
- Multi-provider STT
