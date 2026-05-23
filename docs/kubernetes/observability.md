# Kubernetes Observability

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

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
