# Helm Values Reference

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

Common values:

| Value | Purpose |
| --- | --- |
| `image.repository` | IronRoot server image repository |
| `image.tag` | Image tag; defaults to chart `appVersion` |
| `replicaCount` | Server replica count |
| `service.type` | Defaults to `ClusterIP` |
| `server.port` | API port, default `8443` |
| `persistence.enabled` | Enables SQLite PVC |
| `persistence.size` | SQLite PVC size |
| `pki.existingSecret` | Secret with `root-ca.crt`, `ca-chain.crt`, `intermediate.crt`, and `intermediate.key` |
| `tls.existingSecret` | TLS Secret for API serving cert |
| `config.telemetry.enabled` | Enables OpenTelemetry export |
| `config.telemetry.endpoint` | OTLP endpoint |
| `ingress.enabled` | Enables Ingress |
| `networkPolicy.enabled` | Enables NetworkPolicy |
| `serviceMonitor.enabled` | Enables Prometheus Operator ServiceMonitor |

Security defaults:

- `runAsNonRoot: true`
- `readOnlyRootFilesystem: true`
- `allowPrivilegeEscalation: false`
- `capabilities.drop: ["ALL"]`
- `seccompProfile.type: RuntimeDefault`

Example values live in `deploy/helm/ironroot/examples`.

