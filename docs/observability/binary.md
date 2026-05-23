# Binary Observability

For binary deployments, configure telemetry in `/etc/ironroot/config.yaml` or with environment variables in the systemd unit.

```ini
[Service]
Environment=OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
Environment=OTEL_EXPORTER_OTLP_PROTOCOL=grpc
Environment=OTEL_SERVICE_NAME=ironroot
Environment=OTEL_RESOURCE_ATTRIBUTES=deployment.environment=production
```

Scrape `/metrics` over the API listener. Forward JSON logs from journald or `/var/log/ironroot/` into Loki or your log pipeline.

When troubleshooting, check collector reachability first, then verify IronRoot logs contain trace IDs and Prometheus can scrape `/metrics`.
