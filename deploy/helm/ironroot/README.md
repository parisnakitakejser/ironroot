# IronRoot Helm Chart

This chart deploys `ironroot-server` into Kubernetes with secure defaults for internal PKI operations.

## Install

```bash
helm install ironroot oci://ghcr.io/parisnakitakejser/charts/ironroot \
  --version 0.1.0-rc.1 \
  --namespace ironroot \
  --create-namespace \
  --set pki.existingSecret=ironroot-ca \
  --set tls.existingSecret=ironroot-api-tls
```

## Upgrade

```bash
helm upgrade ironroot oci://ghcr.io/parisnakitakejser/charts/ironroot \
  --version 0.1.0 \
  --namespace ironroot \
  -f values.yaml
```

## Uninstall

```bash
helm uninstall ironroot --namespace ironroot
```

## Values

| Value | Default | Description |
| --- | --- | --- |
| `image.repository` | `ghcr.io/parisnakitakejser/ironroot` | Container image repository |
| `image.tag` | `""` | Image tag, defaults to chart `appVersion` |
| `image.pullPolicy` | `IfNotPresent` | Image pull policy |
| `replicaCount` | `1` | Number of server replicas |
| `server.port` | `8443` | API listener port |
| `service.type` | `ClusterIP` | Kubernetes Service type |
| `service.port` | `8443` | Service port |
| `persistence.enabled` | `true` | Enable SQLite PVC |
| `persistence.size` | `1Gi` | SQLite PVC size |
| `pki.existingSecret` | `""` | Existing Secret containing CA material |
| `pki.mountPath` | `/pki` | CA material mount path |
| `tls.enabled` | `true` | Enable API TLS file configuration |
| `tls.existingSecret` | `""` | Existing TLS Secret for API serving cert |
| `config.database.type` | `sqlite` | Database type, `sqlite` now and `postgres` later |
| `config.telemetry.enabled` | `false` | Enable OpenTelemetry export |
| `config.telemetry.endpoint` | `""` | OTLP endpoint |
| `ingress.enabled` | `false` | Create Ingress |
| `networkPolicy.enabled` | `false` | Create NetworkPolicy |
| `serviceMonitor.enabled` | `false` | Create Prometheus Operator ServiceMonitor |
| `securityContext.readOnlyRootFilesystem` | `true` | Use read-only root filesystem |
| `securityContext.capabilities.drop` | `["ALL"]` | Drop Linux capabilities |

## SQLite

```bash
helm install ironroot ./deploy/helm/ironroot \
  --set pki.existingSecret=ironroot-ca \
  --set tls.existingSecret=ironroot-api-tls \
  --set persistence.enabled=true
```

## PostgreSQL future configuration

The application backend is SQLite-first today. The values are reserved so deployments can keep a stable shape when PostgreSQL support lands:

```bash
helm install ironroot ./deploy/helm/ironroot -f examples/postgres-values.yaml
```

## OpenTelemetry

```bash
helm upgrade --install ironroot ./deploy/helm/ironroot -f examples/otel-values.yaml
```

## Ingress

```bash
helm upgrade --install ironroot ./deploy/helm/ironroot -f examples/ingress-values.yaml
```

## TLS Secret

```bash
kubectl create secret tls ironroot-api-tls \
  --cert=tls.crt \
  --key=tls.key \
  --namespace ironroot
```

## CA Material Secret

```bash
kubectl create secret generic ironroot-ca \
  --from-file=root-ca.crt \
  --from-file=ca-chain.crt \
  --from-file=intermediate.crt \
  --from-file=intermediate.key \
  --namespace ironroot
```

Never put the offline Root CA private key in this Secret.

## Security Notes

- Runs as non-root by default.
- Drops all Linux capabilities by default.
- Uses a read-only root filesystem by default.
- Mounts CA material from Kubernetes Secret.
- Uses ClusterIP by default.
- NetworkPolicy is available but disabled by default to avoid breaking first installs.

## Airgap Notes

Mirror both the image and the chart:

```bash
skopeo copy docker://ghcr.io/parisnakitakejser/ironroot:v0.1.0-rc.1 docker://registry.internal/ironroot:v0.1.0-rc.1
helm pull oci://ghcr.io/parisnakitakejser/charts/ironroot --version 0.1.0-rc.1
helm push ironroot-0.1.0-rc.1.tgz oci://registry.internal/charts
```
