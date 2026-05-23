# Prometheus

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

IronRoot exposes Prometheus metrics at `/metrics` when `telemetry.prometheus.enabled` is true.

Binary scrape example:

```yaml
scrape_configs:
  - job_name: ironroot
    scheme: https
    metrics_path: /metrics
    static_configs:
      - targets: ["ironroot.internal:8443"]
```

Kubernetes users can enable `serviceMonitor.enabled=true` in the Helm chart. Prometheus Operator discovers the ServiceMonitor based on namespace and label selectors.

Retention should match operational needs. Certificate lifecycle trends are useful over weeks or months; high-cardinality request labels should remain bounded to method, route, and status code.

Air-gapped environments should mirror Prometheus images and keep rules/dashboards in GitOps or offline package repositories.
