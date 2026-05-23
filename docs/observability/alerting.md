# Alerting

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

Alerts should focus on operator action. Avoid paging on one-off failures unless they indicate trust compromise or control-plane outage.

Recommended alerts:

| Alert | Severity | Example PromQL | Remediation |
| --- | --- | --- | --- |
| API unavailable | critical | `up{job="ironroot"} == 0` | Check pod/process health, TLS, Service, and network policy. |
| High API latency | medium | `histogram_quantile(0.95, rate(pki_api_request_duration_seconds_bucket[5m])) > 1` | Inspect traces for slow DB or CA spans. |
| Elevated 5xx errors | high | `rate(pki_api_request_failures_total[5m]) > 0.05` | Check server logs and recent config changes. |
| Enrollment attack pattern | high | `increase(pki_bootstrap_token_validation_failures_total[15m]) > 20` | Revoke exposed tokens and review source IPs. |
| Excessive revocations | high | `increase(pki_certificates_revoked_total[1h]) > 10` | Confirm whether this is planned rotation or incident response. |
| DB failures | high | `rate(pki_database_errors_total[5m]) > 0` | Check storage permissions, disk space, and SQLite PVC health. |
| Telemetry export failures | medium | `rate(pki_otel_export_failures_total[5m]) > 0` | Check collector endpoint, TLS, and network policy. |
| Critical security-check failure | critical | `increase(pki_security_check_results_total{severity="critical",status="fail"}[1h]) > 0` | Run `ironroot-admin security-check` and apply remediation. |

Also monitor Root CA and Intermediate CA expiry through security-check reports until dedicated CA expiry gauges are added.
