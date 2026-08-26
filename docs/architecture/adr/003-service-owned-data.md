# ADR 003 — Service-Owned Data and Migrations

**Status:** Accepted  
**Date:** 2026-08-26

## Decision

- Each microservice owns its **business domain** and, where persistence is required, its **data and schema**.
- Each persistent service owns its **migration files**, colocated under `services/<service>/migrations/`.
- There is **no** root-level `migrations/` directory.
- Only services that need persistence get a `migrations/` directory.
- Logical ownership is independent of physical database topology.

## Alternatives

1. **Shared database + shared migrations** at repo root
2. **Service-owned schemas/migrations** (chosen)
3. **One physical database server per service from day one**

## Rationale

- Clear ownership: the team/service that writes the domain also evolves the schema
- Prevents cross-service joins and hidden coupling through a shared schema
- Migrations version with the service that needs them for deploy
- Supports eventual independent scaling and storage choices per domain

## Logical vs physical infrastructure

Logical rule (stable):

> The service owns the data and schema for its domain.

Physical layout (flexible):

- Development may use one PostgreSQL instance with separate databases/schemas (`call_db`, `agent_db`, `tenant_db`, …)
- Production may keep that layout or split instances later
- Changing physical topology does **not** move migration ownership

## Current bootstrap application

- `call-service` includes `migrations/` as the first service expected to own durable call data
- `api-gateway`, `voice-gateway`, and `ai-orchestrator` have **no** migrations yet (no persistence implemented or required at bootstrap)
- No migration tooling, SQL, or database access is implemented yet

## Benefits and tradeoffs

| Benefits | Tradeoffs |
|----------|-----------|
| Explicit boundaries | Cross-domain queries require APIs/events, not SQL joins |
| Safer independent deploys | Eventual consistency across services |
| Schema changes reviewable with service PRs | Duplicate infrastructure patterns unless carefully shared via `pkg/` |

Database access and migration runners will be added only when a service’s persistence layer is intentionally implemented.
