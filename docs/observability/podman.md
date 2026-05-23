# Podman Observability

Run IronRoot with OTEL variables pointed at a local collector:

```bash
podman run --rm \
  -p 8443:8443 \
  -e OTEL_EXPORTER_OTLP_ENDPOINT=http://host.containers.internal:4317 \
  -e OTEL_EXPORTER_OTLP_PROTOCOL=grpc \
  -e OTEL_SERVICE_NAME=ironroot \
  -v ./configs:/config:ro,Z \
  -v ./data:/data:Z \
  -v ./pki:/pki:ro,Z \
  ironroot:dev
```

Use rootless Podman where possible. Mount config, data, and PKI material instead of baking them into the image. On SELinux hosts, use `:Z` labels so the container can read the mounted files.
