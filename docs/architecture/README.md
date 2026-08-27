# Architecture

High-level architecture for the AI Voice Support Platform.

> **Phase 1 media path is implemented:** browser ↔ LiveKit ↔ `voice-gateway` (session tokens, room bot, verification tone).  
> **Phase 2 STT is implemented:** gateway decodes browser audio and streams transcripts from Deepgram.  
> **Phase 3 AI orchestrator is implemented:** final transcripts → `ai-orchestrator` → streaming LLM replies (logs / HTTP SSE). TTS remains Phase 4.

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
Voice Gateway  ← Phase 1–3 (tokens, subscribe, tone, STT, transcript handoff)
  │
  ├── Deepgram STT (Phase 2, when configured)
  │
  ▼
AI Orchestrator  ← Phase 3 (conversation context + LLM)
  │
  ├── LLM (OpenAI-compatible; Gemini default)
  ├── Text-to-Speech          ← Phase 4
  ├── RAG / knowledge        ← later
  ├── Tool calling            ← later
  └── Human escalation        ← later
  │
  ▼
Caller (spoken reply after Phase 4)
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
| **voice-gateway** | Media-plane adapter: LiveKit sessions, PCM decode, STT, orchestrator handoff | No (real-time bridge) | Phase 1–3 runtime |
| **ai-orchestrator** | Conversation control plane: context, LLM streaming, future RAG/tools/escalation | In-memory session state (Phase 3); durable ownership TBD | Phase 3 runtime |
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

Cloud provider is intentionally unlocked; Terraform will isolate provider-specific modules when deployment starts (ADR 004). Docker is the primary container toolchain (ADR 005). LiveKit + Phase-1 token ownership: ADR 006. Phase-2 STT: ADR 007. Phase-3 orchestrator: ADR 008.

## Local LiveKit

Phase 1 local SFU setup: [livekit-local.md](livekit-local.md).

## Phase 2 STT

Streaming speech-to-text verification: [stt-phase2.md](stt-phase2.md).

## Phase 3 AI Orchestrator

Conversation + LLM verification: [orchestrator-phase3.md](orchestrator-phase3.md).

## ADRs

See [adr/README.md](adr/README.md).
