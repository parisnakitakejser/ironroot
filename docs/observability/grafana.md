# Grafana

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

Grafana should combine Prometheus metrics, Tempo traces, and Loki logs.

Recommended dashboards:

- API overview
- Certificate operations
- Enrollment activity
- Security posture
- OpenTelemetry health
- Kubernetes runtime
- Podman runtime
- Airgap operational health

Example JSON dashboards live in `examples/grafana/`. They are intentionally minimal so operators can adapt labels and data source names.

Useful panels:

- API requests per second.
- p95 request latency.
- Failed enrollments and token validation failures.
- Certificates issued, renewed, revoked, active, and expiring.
- Security-check failures by severity.
- DB latency and DB errors.
- Telemetry export failures.
