# Configuration

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

IronRoot reads YAML config and environment variables. The default path is `configs/server.yaml`; containers use `/config/config.yaml`.

Important settings:

- `database.driver`: `sqlite` today, PostgreSQL later
- `database.dsn`: SQLite database path
- `pki.default_lifetime`: default issued certificate lifetime, normally `2160h` for 90 days
- `pki.renew_before`: renewal window, normally `720h` for 30 days
- `telemetry.enabled`: enables OTLP export
- `telemetry.otlp_endpoint`: collector endpoint
- `telemetry.otlp_protocol`: `grpc` or `http`

