# OpenTelemetry Architecture

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

Observability matters in PKI because certificate operations are security-sensitive operational events. Operators need to answer who enrolled, who requested a certificate, what issuer signed it, whether renewals are succeeding, and whether failures are clustered around a service or environment.

## Trace Propagation

```mermaid
sequenceDiagram
  participant CLI as Client CLI
  participant API as IronRoot API
  participant DB as Database
  participant CA as Intermediate CA
  participant Collector as OTEL Collector
  CLI->>CLI: root command span
  CLI->>API: HTTP request with traceparent
  API->>API: server span continues trace
  API->>DB: database operation span
  API->>CA: CSR signing span
  CLI-->>Collector: command traces
  API-->>Collector: server traces and metrics
```

## Telemetry Architecture

```mermaid
flowchart LR
  Client[Client/Admin CLI] -->|OTLP traces/metrics| Collector[OpenTelemetry Collector]
  Server[IronRoot API Server] -->|OTLP traces/metrics/log context| Collector
  Collector --> Backend[Tracing, metrics, logs backend]
  Server --> Audit[(Audit log storage)]
```

The audit log is the durable security record. Telemetry is operational visibility and should not contain private keys, raw bootstrap tokens, or secret values.

## OTLP Export Flow

```mermaid
flowchart TD
  Config[telemetry config] --> SDK[OpenTelemetry SDK]
  SDK --> Exporter[OTLP gRPC or HTTP exporter]
  Exporter --> Collector[OpenTelemetry Collector]
  Collector --> Backend[Tempo, Jaeger, Prometheus, or vendor backend]
```

Client commands create root spans. Server requests continue traces through W3C Trace Context. Security-check emits per-category spans and result metrics by severity and status.
