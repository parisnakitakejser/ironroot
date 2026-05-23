# TLS Certificate Usage

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

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
