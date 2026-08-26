# CI / CD (planned)

Path-based GitHub Actions will live here.

**Not implemented yet.** Workflows will be added when services have buildable code.

Intended direction:

```
GitHub → GitHub Actions (path filters) → Docker build → Docker Hub → Kubernetes
```

Example filters (future):

- `services/ai-orchestrator/**` → ai-orchestrator test/build/deploy
- `services/call-service/migrations/**` → treated as call-service change
- `pkg/telemetry/**` → rebuild/test services that depend on telemetry
