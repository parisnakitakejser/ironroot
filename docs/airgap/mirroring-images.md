# Mirroring Container Images

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

Mirror images into an internal registry before deployment:

```bash
skopeo copy docker://ghcr.io/OWNER/ironroot:v0.1.0 \
  docker://registry.internal/ironroot:v0.1.0
```

Also mirror OpenTelemetry Collector, Prometheus, Grafana, Tempo, Loki, and any demo images required by your environment.
