# ADR 001 — Monorepo vs Multiple Repositories

**Status:** Accepted  
**Date:** 2026-08-26

## Decision

Use a **single monorepo** for all services, shared infrastructure packages, deployment skeletons, and docs.

Do **not** create a separate Git repository per microservice.

## Alternatives

1. **Multi-repo** — one repository per service/package
2. **Monorepo** — single repository, independent deployables (chosen)
3. **Hybrid** — platform repo + isolated experiment repos

## Why monorepo

- One place for architecture docs, ADRs, and conventions
- Path-based CI can still target individual services
- Shared `pkg/` infrastructure can evolve without versioning a matrix of repos early
- Lower coordination overhead for a small team / personal project
- Easier atomic refactors across service boundaries when contracts change

## Tradeoffs

| Upside | Downside |
|--------|----------|
| Unified visibility and docs | Repo can grow large over time |
| Simpler local checkout | Requires discipline so services stay decoupled |
| Shared CI conventions | Path filters and ownership must be maintained |

## Independent deployability

Monorepo ≠ monolith deploy:

- Each service has its own `go.mod`, `cmd/`, and (when needed) `migrations/`
- Services must not share business/domain modules
- Future CI builds and deploys only services affected by a change
- Container images and Helm releases remain per-service

Independently buildable and deployable remains a hard requirement regardless of repository layout.
