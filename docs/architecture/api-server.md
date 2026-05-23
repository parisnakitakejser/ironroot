# IronRoot API Server

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

The API server is the online control plane for enrollment, certificate issuance, renewal, revocation metadata, audit logging, and OpenTelemetry instrumentation.

## Responsibilities

- Expose REST endpoints for health, readiness, CA chain, enrollment, certificate lifecycle, and audit reads.
- Validate bootstrap token, hostname, and machine ID during enrollment.
- Accept CSRs from clients and sign them with the Intermediate CA.
- Store certificate metadata, enrollment records, revocation records, audit logs, bootstrap token hashes, and CA generation metadata.
- Emit traces, metrics, and structured logs.

## Request Lifecycle

```mermaid
sequenceDiagram
  participant Client
  participant API
  participant DB
  participant CA
  participant Audit
  participant OTel
  Client->>API: request with traceparent
  API->>OTel: continue server span
  API->>DB: validate enrollment or token
  API->>CA: sign CSR when needed
  API->>DB: write metadata
  API->>Audit: write audit event
  API-->>Client: response
```

## Deployment Types

| Type | Where it runs | Security implications |
| --- | --- | --- |
| Binary | Dedicated host or VM | Strong host filesystem control; you own systemd and backups |
| Podman | Rootless container on host | Immutable image with mounted `/config`, `/data`, `/pki` |
| Kubernetes | Deployment with Secret, ConfigMap, PVC | Requires tight Secret RBAC, securityContext, NetworkPolicy, and internal Service design |

## Audit Flow

```mermaid
flowchart LR
  Action[Enrollment / issuance / renewal / revocation] --> API[IronRoot API]
  API --> Audit[Audit logger]
  Audit --> DB[(audit_logs table)]
  API --> Logs[Structured logs with trace_id/span_id]
```

Audit logging is not optional behavior in the current server write path. Retention controls are future work, so operators should back up and rotate database storage according to local policy.

