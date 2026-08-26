# AI Voice Support Platform

Multi-tenant AI Voice Support Platform for real-time conversational support across customer support, HR, finance, and IT/helpdesk.

Organizations will eventually configure AI voice agents that converse with users, retrieve organization-specific knowledge, invoke backend tools, and escalate to human agents.

> **Status:** Phase 1–2 runnable locally (LiveKit + `voice-gateway` + browser prototype + optional Deepgram STT). AI orchestrator, TTS, and later infrastructure are not implemented yet.

---

## Currently implemented

This repository currently contains:

- Monorepo skeleton and directory layout
- Independent Go modules per deployable service
- Makefile targets for tests, quality gate, LiveKit, voice-gateway, and the browser prototype
- Architecture overview and Architecture Decision Records (ADRs), including LiveKit (ADR 006)
- **Phase 1:** local LiveKit (Docker), `voice-gateway` session tokens + room bot + verification tone, `apps/voice-prototype` browser client
- **Phase 2:** streaming STT — Opus→PCM in gateway, Deepgram WebSocket, partial/final transcript logs ([docs](docs/architecture/stt-phase2.md), [ADR 007](docs/architecture/adr/007-deepgram-stt-in-voice-gateway.md))

LLM/TTS, Kafka, Redis, PostgreSQL runtime, Kubernetes, and cloud deploy are still ahead.

## Planned architecture (not implemented)

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
| `voice-gateway` | Bridge between LiveKit / media plane and AI orchestration | Stateless (no migrations yet) |
| `ai-orchestrator` | Conversation loop: STT → LLM/RAG/tools → TTS, escalation | TBD when domain state is defined |
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
| Language | Go, idiomatic modules | voice-gateway implemented; others stubs |
| Build | Conventional Go modules + Makefile | Active |
| Realtime media | LiveKit (browser WebRTC) | Phase 1 local Docker + gateway |
| STT | Deepgram (streaming, gateway) | Phase 2 (optional via env) |
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

Phase 1–2 local stack:

```bash
make livekit-up
make run-voice-gateway   # loads .env; set DEEPGRAM_API_KEY for STT
make run-voice-prototype   # http://127.0.0.1:5173
```

Verification: [livekit-local.md](docs/architecture/livekit-local.md), [stt-phase2.md](docs/architecture/stt-phase2.md).

```bash
make test    # all service modules
make quality # custom quality gate
```

## Engineering principles

Prefer simple solutions, explicit service boundaries, service-owned data, dependency injection over globals, and incremental delivery. Do not introduce technologies only because they look impressive—verify each architectural layer before building the next.

## Documentation

- [Architecture overview](docs/architecture/README.md)
- [ADR index](docs/architecture/adr/README.md)
