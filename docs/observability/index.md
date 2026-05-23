# OpenTelemetry

IronRoot is OpenTelemetry-first.

```mermaid
sequenceDiagram
  participant CLI as pki-client request-cert
  participant API as POST /v1/certificates/request
  participant DB as SQLite
  participant CA as Intermediate CA
  CLI->>API: traceparent
  API->>DB: validate enrollment
  API->>CA: validate CSR and sign certificate
  API->>DB: store metadata
  API-->>CLI: certificate chain
```

Server-side telemetry includes endpoint tracing, request duration, status code metrics, and structured logs with trace and span IDs.

Client-side telemetry creates a root span per command, records command duration, tracks errors, and propagates trace context to the server.

Supported configuration includes service name, service version, deployment environment, OTLP endpoint, OTLP protocol, and sampling ratio.

