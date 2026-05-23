# Container Mirroring

Mirror IronRoot and dependency images into an internal registry.

```bash
skopeo copy docker://ghcr.io/OWNER/ironroot:v0.1.0 \
  docker://registry.internal/ironroot:v0.1.0
```
