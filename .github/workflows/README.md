# CI / CD

## Implemented

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| [`branch-name.yml`](branch-name.yml) | `push` (branches) | Enforce branch naming convention (`feature/…`, `bugfix/…`, etc.) |
| [`pr-quality.yml`](pr-quality.yml) | `pull_request` | Custom Go AST quality gate (`make quality-report`) |

The quality workflow checks out full history, compares against `origin/<base_ref>`, uploads JSON/SARIF artifacts, writes a PR job summary, and uploads SARIF to code scanning when permitted.

## Planned

Path-based build/test/deploy workflows will be added when services have buildable application code.

Intended direction:

```
GitHub → GitHub Actions (path filters) → Docker build → Docker Hub → Kubernetes
```

Example filters (future):

- `services/ai-orchestrator/**` → ai-orchestrator test/build/deploy
- `services/call-service/migrations/**` → treated as call-service change
- `pkg/telemetry/**` → rebuild/test services that depend on telemetry
