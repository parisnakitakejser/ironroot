# Kubernetes Observability

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

IronRoot supports OpenTelemetry Collector integration, Prometheus scraping, and optional ServiceMonitor resources in Kubernetes.

Enable telemetry and ServiceMonitor in Helm values:

```yaml
config:
  telemetry:
    enabled: true
    endpoint: opentelemetry-collector.observability.svc:4317
serviceMonitor:
  enabled: true
```

For the full observability model, see the Observability section.
