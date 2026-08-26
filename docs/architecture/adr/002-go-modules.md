# ADR 002 — Go Modules vs Bazel/Nx

**Status:** Accepted  
**Date:** 2026-08-26

## Decision

Use **conventional Go modules**: one `go.mod` per independently deployable service.

Do **not** adopt Bazel or Nx at this stage.

## Alternatives

1. **Conventional Go modules** (chosen)
2. **Bazel** — hermetic, large-scale build graph
3. **Nx** — JS-centric monorepo tooling (poor fit as primary Go build system)
4. **Single root Go module** — all services in one module

## Why conventional modules

- Idiomatic Go; low onboarding cost
- Natural independent versioning and dependency graphs per service
- Avoids early investment in complex build tooling before code exists
- Aligns with “prefer simple solutions over premature abstractions”

A single root module was rejected because it encourages accidental cross-service coupling and blurs deployable boundaries.

## Tradeoffs

| Upside | Downside |
|--------|----------|
| Simple, standard tooling | Cross-cutting `pkg` changes need explicit consumer updates |
| Clear service isolation | No advanced remote caching / hermetic builds yet |
| Easy local `go test` / `go build` | Path-based CI must be maintained manually |

## When a heavier build system might be justified

Revisit Bazel (or similar) only if:

- Build/test times become a clear bottleneck at scale
- Hermetic reproducibility is a hard compliance requirement
- A large number of services share complex generated artifacts

Until then, Makefile + Go modules + GitHub Actions path filters are sufficient.
