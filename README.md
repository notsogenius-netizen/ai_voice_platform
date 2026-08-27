# AI Voice Support Platform

Multi-tenant AI Voice Support Platform for real-time conversational support across customer support, HR, finance, and IT/helpdesk.

Organizations will eventually configure AI voice agents that converse with users, retrieve organization-specific knowledge, invoke backend tools, and escalate to human agents.

> **Status:** Phase 1–4 runnable locally (LiveKit + `voice-gateway` + browser prototype + optional Deepgram STT/TTS + `ai-orchestrator` LLM turns). Later infrastructure (persistence, Redis, tenancy, RAG) is not implemented yet.

---

## Currently implemented

This repository currently contains:

- Monorepo skeleton and directory layout
- Independent Go modules per deployable service
- Makefile targets for tests, quality gate, LiveKit, voice-gateway, ai-orchestrator, and the browser prototype
- Architecture overview and Architecture Decision Records (ADRs), including LiveKit (ADR 006), STT (ADR 007), orchestrator (ADR 008), and TTS design (ADR 009)
- **Phase 1:** local LiveKit (Docker), `voice-gateway` session tokens + room bot + verification tone, `apps/voice-prototype` browser client
- **Phase 2:** streaming STT — Opus→PCM in gateway, Deepgram WebSocket, partial/final transcript logs ([docs](docs/architecture/stt-phase2.md), [ADR 007](docs/architecture/adr/007-deepgram-stt-in-voice-gateway.md))
- **Phase 3:** `ai-orchestrator` — in-memory conversation state, OpenAI-compatible streaming LLM (Gemini default), turn API; gateway forwards final transcripts ([docs](docs/architecture/orchestrator-phase3.md), [ADR 008](docs/architecture/adr/008-ai-orchestrator-phase3.md))
- **Phase 4:** Deepgram Speak TTS in `voice-gateway` — Ogg Opus publish to LiveKit, sentence streaming, barge-in ([docs](docs/architecture/tts-phase4.md), [ADR 009](docs/architecture/adr/009-deepgram-tts-in-voice-gateway.md))

Kafka, Redis, PostgreSQL runtime, Kubernetes, and cloud deploy are still ahead.

## Planned architecture (not fully implemented)

The eventual platform is intended to demonstrate:

- Go backend engineering and microservice boundaries
- Real-time voice (WebRTC / SIP via LiveKit)
- AI/LLM orchestration, STT/TTS, RAG, and tool calling
- Event-driven design (e.g. Kafka), caching (Redis), persistence (PostgreSQL)
- Reliability patterns, observability, containers, Kubernetes, and Terraform-based IaC

See [docs/architecture/README.md](docs/architecture/README.md) for the planned flow and service responsibilities.

---

## Why a monorepo?

Services stay **independently buildable and deployable**, while living in one repository so shared infrastructure packages, ADRs, and path-based CI can evolve together without a multi-repo coordination tax.

Decision details: [ADR 001](docs/architecture/adr/001-monorepo.md).

## Planned services

| Service | Role (planned) | Persistence (planned) |
|---------|----------------|------------------------|
| `api-gateway` | External HTTP/API edge, routing, auth boundary | Stateless (no migrations yet) |
| `voice-gateway` | Bridge between LiveKit / media plane and AI orchestration (STT + Phase 4 TTS) | Stateless (no migrations yet) |
| `ai-orchestrator` | Conversation loop: LLM (+ future RAG/tools), escalation | In-memory sessions today; durable state later |
| `call-service` | Call lifecycle, metadata, and call-domain persistence | Owns schema/migrations |

Each service has its own `go.mod` and is expected to remain independently deployable. There is **no shared business/domain module**.

## Service ownership

- Business logic for a domain lives **inside the owning service**.
- Services must not import each other's `internal/` packages.
- Cross-service communication will use explicit APIs/events (to be designed later), not shared domain libraries.

## Database / migration ownership

- A service that owns persistent data owns its **schema and migrations**.
- Migration files live under that service (e.g. `services/call-service/migrations/`).
- There is **no** root-level `migrations/` directory.
- Logical ownership ≠ physical topology: one PostgreSQL instance may host multiple databases/schemas in development; each service still owns its schema.

Decision details: [ADR 003](docs/architecture/adr/003-service-owned-data.md).

Only `call-service` has a `migrations/` directory today, as the first service expected to own durable call data. Other services will gain migrations only when they own persistent state.

## Technology direction

| Area | Direction | Status |
|------|-----------|--------|
| Language | Go, idiomatic modules | voice-gateway + ai-orchestrator implemented; others stubs |
| Build | Conventional Go modules + Makefile | Active |
| Realtime media | LiveKit (browser WebRTC) | Phase 1 local Docker + gateway |
| STT | Deepgram (streaming, gateway) | Phase 2 (optional via env) |
| TTS | Deepgram Speak → LiveKit (gateway, Ogg Opus) | Phase 4 (optional via env) |
| LLM | OpenAI-compatible API (Gemini default) | Phase 3 (optional via env) |
| Containers | Docker / OCI; Docker Hub as initial registry | LiveKit Compose only so far |
| Orchestration | Kubernetes + Helm (planned) | Directories only |
| IaC | Terraform, cloud-agnostic layout | Directories only |
| Cloud | Undecided; AWS currently likely for cost | Not locked ([ADR 004](docs/architecture/adr/004-cloud-provider.md)) |
| Messaging / cache / DB | Kafka, Redis, PostgreSQL (planned) | Not implemented |
| CI | GitHub Actions with path filters (planned) | Quality workflow present; deploy later |

Build system choice: [ADR 002](docs/architecture/adr/002-go-modules.md).  
Container tooling: [ADR 005](docs/architecture/adr/005-docker-vs-podman.md).  
LiveKit / tokens: [ADR 006](docs/architecture/adr/006-livekit-media-and-tokens.md).  
Phase 2 STT: [ADR 007](docs/architecture/adr/007-deepgram-stt-in-voice-gateway.md).  
Phase 3 orchestrator: [ADR 008](docs/architecture/adr/008-ai-orchestrator-phase3.md).  
Phase 4 TTS: [ADR 009](docs/architecture/adr/009-deepgram-tts-in-voice-gateway.md).

## Deployment direction (planned)

```
GitHub → GitHub Actions (path-based) → Docker build → Docker Hub → Kubernetes
```

Terraform will manage cloud infrastructure once a provider is chosen. No Dockerfiles, Helm charts, Terraform resources, or production CI pipelines exist yet.

## Repository layout

```
services/           # Independently deployable Go services
pkg/                # Shared infrastructure concerns only (no domain logic)
proto/              # Future API / event contracts
deployments/        # docker /, helm /, terraform / (empty placeholders)
docs/architecture/  # Architecture overview + ADRs
.github/workflows/  # Future path-based CI
Makefile
README.md
```

`pkg/` is reserved for reusable platform concerns (Kafka/Redis/Postgres clients, logging, telemetry). It must **not** become a dumping ground for business domains (`user`, `call`, `agent`, etc.).

## Local development

Phase 1–4 local stack:

```bash
make livekit-up
make run-ai-orchestrator   # loads .env; set LLM_API_KEY for LLM turns
make run-voice-gateway     # loads .env; DEEPGRAM_API_KEY + AI_ORCHESTRATOR_URL for STT→LLM→TTS
make run-voice-prototype   # http://127.0.0.1:5173
```

Verification: [livekit-local.md](docs/architecture/livekit-local.md), [stt-phase2.md](docs/architecture/stt-phase2.md), [orchestrator-phase3.md](docs/architecture/orchestrator-phase3.md), [tts-phase4.md](docs/architecture/tts-phase4.md).

```bash
make test    # all service modules
make quality # custom quality gate
```

## Engineering principles

Prefer simple solutions, explicit service boundaries, service-owned data, dependency injection over globals, and incremental delivery. Do not introduce technologies only because they look impressive—verify each architectural layer before building the next.

## Documentation

- [Architecture overview](docs/architecture/README.md)
- [ADR index](docs/architecture/adr/README.md)
- [Phase 2 STT](docs/architecture/stt-phase2.md)
- [Phase 3 AI Orchestrator](docs/architecture/orchestrator-phase3.md)
- [Phase 4 TTS](docs/architecture/tts-phase4.md)
