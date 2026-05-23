# Podman

Build the image:

```bash
make container-build
```

Run locally:

```bash
podman run --rm -p 8443:8443 \
  -v ./configs:/config:ro \
  -v ./data:/data \
  -v ./pki:/pki:ro \
  localhost/ironroot:dev
```

The container runs as a non-root user and expects config at `/config/config.yaml`, data at `/data`, and CA material at `/pki`.

