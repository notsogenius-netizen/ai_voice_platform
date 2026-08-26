# ADR 005 — Docker vs Podman

**Status:** Accepted  
**Date:** 2026-08-26

## Decision

- **Docker** is the primary container tooling for this project
- **Docker Hub** is the initial container registry target
- **Podman** is not used as the primary project tooling
- Application images and runtime assumptions should remain **OCI-compatible**, not tightly coupled to Docker-only behavior

## Alternatives

1. Docker + Docker Hub (chosen)
2. Podman as primary CLI/runtime
3. Provider-specific build services only (no local Docker workflow)

## Rationale

- Docker remains the most common local and CI container workflow
- Docker Compose is a familiar path for future multi-service local development
- Docker Hub is a simple starting registry for a personal/project-scale pipeline
- Staying OCI-compatible avoids accidental dependence on Docker-specific runtime quirks and keeps future runtimes (containerd, CRI-O, etc.) viable

## Tradeoffs

| Upside | Downside |
|--------|----------|
| Broad ecosystem and docs | Docker Desktop licensing/context varies by environment |
| Straightforward CI (`docker build`) | Not rootless-by-default like Podman |
| Compose for local multi-service later | Registry may move (GHCR, ECR, etc.) later |

## Bootstrap scope

No Dockerfiles or Compose files are created yet. Containerization will be added when the first real service implementation lands. Directory placeholder: `deployments/docker/`.
