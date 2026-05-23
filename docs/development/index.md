# Development

Common commands:

```bash
make build
make test
make test-e2e
make lint
make docs-build
make container-build
```

The first backend is SQLite. Keep repository interfaces narrow so PostgreSQL can be added without changing API or CLI behavior.

Contributors should prefer clear tests around PKI behavior, enrollment validation, storage contracts, and telemetry propagation.

