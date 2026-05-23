# Metrics

IronRoot metrics describe PKI operations, API health, security posture, storage latency, and telemetry pipeline health. Metrics are available at `/metrics` when Prometheus export is enabled and can also be exported through OTLP.

## API Metrics

| Metric | Type | Labels | Unit | Meaning |
| --- | --- | --- | --- | --- |
| `pki_api_requests_total` | Counter | `method`, `route`, `status_code` | requests | API traffic volume and endpoint usage. |
| `pki_api_request_duration_seconds` | Histogram | `method`, `route`, `status_code` | seconds | API latency. |
| `pki_api_request_failures_total` | Counter | `method`, `route`, `status_code` | requests | Server-side API failures. |

Good values are steady request rates, low p95 latency, and near-zero 5xx responses. Bad values are latency spikes, request floods, or elevated 5xx responses.

```promql
rate(pki_api_requests_total[5m])
histogram_quantile(0.95, rate(pki_api_request_duration_seconds_bucket[5m]))
rate(pki_api_request_failures_total[5m])
```

Alert on sustained high p95 latency or elevated 5xx rates.

## Enrollment Metrics

| Metric | Type | Labels | Unit | Meaning |
| --- | --- | --- | --- | --- |
| `pki_enrollments_total` | Counter | none | enrollments | Successful enrollment activity. |
| `pki_enrollment_failures_total` | Counter | none | failures | Failed enrollment attempts. |
| `pki_bootstrap_token_validation_failures_total` | Counter | none | failures | Invalid, expired, or revoked bootstrap token attempts. |

Use these metrics to spot onboarding problems and possible token guessing attacks.

```promql
increase(pki_enrollment_failures_total[15m])
increase(pki_bootstrap_token_validation_failures_total[15m])
```

## Certificate Lifecycle Metrics

| Metric | Type | Labels | Unit | Meaning |
| --- | --- | --- | --- | --- |
| `pki_certificates_issued_total` | Counter | `certificate_type`, `issuer`, `status` | certificates | New certificate issuance. |
| `pki_certificates_renewed_total` | Counter | `certificate_type`, `issuer`, `status` | certificates | Renewal activity. |
| `pki_certificates_revoked_total` | Counter | `certificate_type`, `issuer`, `status` | certificates | Revocation activity. |
| `pki_active_certificates_total` | UpDownCounter | `certificate_type`, `issuer`, `status` | certificates | Approximate active inventory. |
| `pki_expiring_certificates_total` | UpDownCounter | `certificate_type`, `issuer`, `status` | certificates | Certificates approaching expiry. |

```promql
increase(pki_certificates_issued_total[24h])
increase(pki_certificates_revoked_total[1h])
pki_active_certificates_total
```

High revocation rates may indicate compromise or automation errors. Expiring certificate growth means renewals are not keeping up.

## Security-Check Metrics

| Metric | Type | Labels | Unit | Meaning |
| --- | --- | --- | --- | --- |
| `pki_security_check_results_total` | Counter | `severity`, `status`, `category` | checks | Security posture checks by outcome. |

Alert when critical or high severity checks fail.

```promql
increase(pki_security_check_results_total{severity=~"critical|high",status="fail"}[1h])
```

## Bootstrap Metrics

| Metric | Type | Labels | Unit | Meaning |
| --- | --- | --- | --- | --- |
| `pki_bootstrap_runs_total` | Counter | none | runs | Bootstrap guide executions. |
| `pki_bootstrap_failures_total` | Counter | none | failures | Failed bootstrap runs. |

Repeated failures usually mean automation is missing required acknowledgements or config paths.

## Database Metrics

| Metric | Type | Labels | Unit | Meaning |
| --- | --- | --- | --- | --- |
| `pki_database_query_duration_seconds` | Histogram | `operation`, `status` | seconds | Storage latency per repository operation. |
| `pki_database_errors_total` | Counter | `operation`, `status` | errors | Storage failures. |

```promql
histogram_quantile(0.95, rate(pki_database_query_duration_seconds_bucket[5m]))
rate(pki_database_errors_total[5m])
```

High DB latency often points to slow persistent storage, overloaded SQLite filesystems, or future PostgreSQL network problems.

## OpenTelemetry Exporter Metrics

| Metric | Type | Labels | Unit | Meaning |
| --- | --- | --- | --- | --- |
| `pki_otel_export_failures_total` | Counter | `operation` | failures | Exporter initialization or export failures. |
| `pki_otel_export_duration_seconds` | Histogram | `operation`, `status` | seconds | Exporter operation latency. |

Failures here mean observability may be degraded even when PKI still works.

## CLI Metrics

| Metric | Type | Labels | Unit | Meaning |
| --- | --- | --- | --- | --- |
| `pki_cli_command_duration_seconds` | Histogram | `command`, `status` | seconds | CLI operation duration. |
| `pki_cli_command_errors_total` | Counter | `command`, `status` | errors | CLI command failures. |
| `pki_cli_command_results_total` | Counter | `command`, `status` | commands | CLI success and failure count. |

Use CLI metrics to identify slow enrollment, renewal friction, and operator workflow failures.

## Runtime Metrics

When runtime metrics are enabled by the collector or Go runtime instrumentation, track memory, goroutines, GC duration, CPU, and open file descriptors. Good values are stable. Bad values are steadily rising memory, long GC pauses, or file descriptor exhaustion.

## Dashboard Guidance

Dashboards should include API health, certificate issuance, renewals, revocations, expiring certificates, failed enrollments, failed token validations, security-check failures, DB latency, and telemetry export health. Example dashboards live in `examples/grafana/`.
