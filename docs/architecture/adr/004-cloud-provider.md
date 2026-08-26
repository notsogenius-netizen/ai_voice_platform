# ADR 004 — Cloud Provider Strategy

**Status:** Accepted  
**Date:** 2026-08-26

## Decision

Do **not** lock the project to AWS, GCP, or Azure at bootstrap.

- Cloud provider selection is **intentionally undecided**
- **AWS is currently the most likely** candidate (personal project; cost sensitivity)
- **Terraform** will be the IaC layer and the primary abstraction boundary
- Cloud-specific modules/resources will be chosen when deployment is actually implemented

## Alternatives

1. Commit to AWS immediately
2. Commit to GCP or Azure immediately
3. Stay provider-agnostic until first deploy (chosen)
4. Multi-cloud from day one

## Rationale

- Premature provider lock-in produces unused Terraform and misleading “architecture”
- Cost and operational simplicity matter more than resume-driven multi-cloud
- Terraform under `deployments/terraform/` can isolate provider-specific modules later
- Keeping layouts cloud-neutral preserves optionality without implementing fake resources now

## Tradeoffs

| Upside | Downside |
|--------|----------|
| No wasted cloud-specific IaC yet | Some deployment details deferred |
| Cost-driven choice when needed | Slightly less “complete” looking infra tree |
| Clear separation via Terraform modules later | Temporary ambiguity for contributors |

## Guidance for later work

- Prefer portable interfaces (OCI images, Kubernetes, managed Postgres/Kafka-like services) before provider APIs
- When a provider is chosen, document it in a follow-up ADR and add Terraform modules under a provider-specific path
- Do not create real cloud resources or AWS/GCP/Azure Terraform resources in this bootstrap
