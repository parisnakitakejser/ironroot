# TLS Certificate Usage

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

Mount generated certificates read-only into containers:

```bash
podman run --rm \
  -v .localdev/certs/demo.local:/certs:ro,Z \
  -p 8444:443 \
  nginx
```

Use:

- `/certs/tls.key` for the private key.
- `/certs/tls.crt` for the issued certificate.
- `/certs/fullchain.crt` for servers that expect the leaf certificate plus chain.
- `/certs/ca-chain.crt` for trust-chain debugging.

Keep `tls.key` out of images. Mount it at runtime from host storage, a secret manager, or a Kubernetes Secret.
