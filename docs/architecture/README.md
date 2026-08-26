# Architecture

High-level architecture for the AI Voice Support Platform.

> **Phase 1 media path is implemented:** browser ↔ LiveKit ↔ `voice-gateway` (session tokens, room bot, verification tone). AI/STT/TTS and later services remain planned.

---

## Problem and scope

Organizations need multi-tenant voice support agents that can:

- Hold real-time conversations with callers
- Use organization-specific knowledge (RAG)
- Invoke backend tools
- Escalate to human agents when needed

Support domains include customer support, HR, finance, and IT/helpdesk.

## Core voice flow

```
Browser (apps/voice-prototype)
  │  WebRTC
  ▼
LiveKit (local Docker)
  │
  ▼
Voice Gateway  ← Phase 1 (tokens, subscribe, verification tone)
  │
  ▼
AI Orchestrator  ← later phases
  │
  ├── Speech-to-Text
  ├── LLM
  ├── Text-to-Speech
  ├── RAG / knowledge retrieval
  ├── Tool calling
  └── Human escalation
  │
  ▼
Caller
```

Supporting capabilities planned around this core:

- Call lifecycle and metadata (`call-service`)
- API edge / tenancy / configuration (`api-gateway`)
- Event bus (Kafka), cache (Redis), relational data (PostgreSQL)
- Analytics / notification workers, recording / object storage
- Observability, Kubernetes, Terraform, cloud deployment

## Service responsibilities

| Service | Responsibility | Owns data? | Status |
|---------|----------------|------------|--------|
| **api-gateway** | External API surface, request routing, future auth/tenancy boundary | No (stateless edge) | Skeleton |
| **voice-gateway** | Media-plane adapter: LiveKit sessions ↔ orchestration signals | No (real-time bridge) | Phase 1 runtime |
| **ai-orchestrator** | Conversation control plane: STT/LLM/TTS/RAG/tools/escalation orchestration | TBD when durable agent/session state is defined | Skeleton |
| **call-service** | Call records, lifecycle state, call-domain queries | Yes — owns schema + migrations | Skeleton |

### Ownership rules

1. Business/domain logic lives in the owning service (`internal/`).
2. No shared domain packages under `pkg/`.
3. Persistent services own their migrations under `services/<name>/migrations/`.
4. Services communicate via explicit contracts (HTTP/gRPC/events)—not by importing each other’s internals.

## Monorepo and modules

- One git repository; **one Go module per deployable service**.
- Services are independently buildable and deployable.
- `pkg/` holds infrastructure helpers only (clients, logging, telemetry)—added when needed, not prematurely.

See ADRs 001–003.

## Data ownership (logical vs physical)

Logical ownership is fixed: the service owns its schema and migrations.

Physical topology can change:

```
PostgreSQL (example local layout — not provisioned yet)
├── call_db      ← call-service
├── agent_db     ← future owning service
└── tenant_db    ← future owning service
```

One instance with multiple databases/schemas is acceptable for development cost; production may split further. Ownership of migrations does not change.

## Shared packages (`pkg/`)

Intended (future) contents:

- `kafka/` — client setup
- `redis/` — client setup
- `postgres/` — connection / pool helpers
- `telemetry/` — OpenTelemetry wiring
- `logger/` — structured logging

Forbidden: domain types or repositories (`pkg/call`, `pkg/agent`, etc.).

## Deployments and CI

```
deployments/
  docker/      # LiveKit Compose (Phase 1); service Dockerfiles later
  helm/        # Kubernetes charts later
  terraform/   # Cloud IaC later (provider undecided)

.github/workflows/  # Path-based GitHub Actions later
```

Intended CI shape:

- Change under `services/<svc>/**` → build/test/deploy that service
- Change under `pkg/**` → rebuild/test services that depend on the changed package
- Migration changes under a service → treated as a change to that service

Cloud provider is intentionally unlocked; Terraform will isolate provider-specific modules when deployment starts (ADR 004). Docker is the primary container toolchain (ADR 005). LiveKit + Phase-1 token ownership: ADR 006.

## Incremental delivery

Implementation proceeds layer by layer. Each step should verify boundaries and ownership before adding the next technology. Prefer production-realistic simplicity over resume-driven complexity.

## Local LiveKit

Phase 1 local SFU setup: [livekit-local.md](livekit-local.md).

## ADRs

See [adr/README.md](adr/README.md).
