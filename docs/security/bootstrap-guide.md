# First-Run Bootstrap Guide

Run the interactive guide before exposing IronRoot to operators or workloads:

```bash
ironroot-admin bootstrap --output-checklist ./ironroot-security-checklist.md
```

For automation:

```bash
ironroot-admin bootstrap \
  --non-interactive \
  --acknowledge-risk \
  --config /config/config.yaml \
  --output-checklist ./ironroot-security-checklist.md
```

The guide walks through:

- Offline Root CA handling
- Intermediate CA handling
- API TLS posture
- server filesystem permissions
- database and CA backups
- audit logging
- OpenTelemetry configuration
- migration and recovery readiness

Non-interactive mode requires `--acknowledge-risk` so CI/CD and air-gapped automation record an explicit security decision.

