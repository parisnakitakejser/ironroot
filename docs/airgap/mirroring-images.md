# Mirroring Container Images

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

Mirror images into an internal registry before deployment:

```bash
skopeo copy docker://ghcr.io/parisnakitakejser/ironroot:v0.1.0 \
  docker://registry.internal/ironroot:v0.1.0
```

Also mirror OpenTelemetry Collector, Prometheus, Grafana, Tempo, Loki, and any demo images required by your environment.
