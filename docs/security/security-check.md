# Security Check

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

`ironroot-admin security-check` inspects the host, IronRoot config, CA material, database posture, API TLS settings, bootstrap token posture, audit logging, telemetry, and runtime environment.

```bash
ironroot-admin security-check --config /config/config.yaml --output table
ironroot-admin security-check --output json
ironroot-admin security-check --output markdown --write-report security-report.md
ironroot-admin security-check --fail-on high
```

## Severity

- `info`: operational context
- `low`: useful hardening signal
- `medium`: should be fixed before production
- `high`: security issue that can expose trust infrastructure
- `critical`: violates a core PKI safety boundary

Exit codes:

- `0`: no failed checks at or above the threshold
- `1`: failed checks at or above the threshold
- `2`: configuration or runtime error

## CI/CD

Use a non-secret development config in CI:

```bash
ironroot-admin security-check \
  --config ./examples/config.dev.yaml \
  --output json \
  --fail-on critical
```

Do not put production private keys, token values, or secret material into CI.

