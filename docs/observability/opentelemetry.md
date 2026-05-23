# OpenTelemetry Configuration

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

IronRoot can export traces and metrics through OTLP gRPC or OTLP HTTP. Prometheus scraping can be enabled at the same time.

```yaml
telemetry:
  enabled: true
  service_name: ironroot
  deployment_environment: production
  traces:
    enabled: true
  metrics:
    enabled: true
  logs:
    enabled: true
  exporter:
    protocol: grpc
    endpoint: otel-collector:4317
    insecure: true
  sampling:
    ratio: 1.0
  prometheus:
    enabled: true
    path: /metrics
```

Supported environment variables:

- `OTEL_EXPORTER_OTLP_ENDPOINT`
- `OTEL_EXPORTER_OTLP_PROTOCOL`
- `OTEL_SERVICE_NAME`
- `OTEL_RESOURCE_ATTRIBUTES`
- `OTEL_TRACES_SAMPLER`
- `OTEL_TRACES_SAMPLER_ARG`

Disable telemetry in small labs by setting `telemetry.enabled: false`. Keep Prometheus enabled if you still want local `/metrics`.
