# Container Mirroring

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

Mirror IronRoot and dependency images into an internal registry.

```bash
skopeo copy docker://ghcr.io/OWNER/ironroot:v0.1.0 \
  docker://registry.internal/ironroot:v0.1.0
```
