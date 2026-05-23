# Container Mirroring

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

Mirror IronRoot and dependency images into an internal registry.

```bash
skopeo copy docker://ghcr.io/OWNER/ironroot:v0.1.0 \
  docker://registry.internal/ironroot:v0.1.0
```
