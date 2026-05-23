# Airgap-First Architecture

Airgap-first means IronRoot can be installed, operated, observed, and recovered without assuming live Internet access from the target environment.

## Deployment Flow

```mermaid
flowchart TD
  Online[Connected build/release environment] --> Mirror[Mirror artifacts]
  Mirror --> Transfer[Approved offline transfer path]
  Transfer --> Registry[Internal registry]
  Registry --> K8s[Kubernetes or Podman host]
  OfflineRoot[Offline Root CA] --> Intermediate[Signed Intermediate CA]
  Intermediate --> K8s
```

## Artifact Mirroring

```mermaid
flowchart LR
  GHCRImage[ghcr.io/OWNER/ironroot] --> InternalImage[internal registry image]
  GHCRChart[ghcr.io/OWNER/charts/ironroot] --> InternalChart[internal chart registry]
  ReleaseTGZ[GitHub Release chart tgz] --> OfflineMedia[offline media]
```

Mirror:

- container image
- Helm chart OCI artifact or `.tgz`
- documentation package if needed
- public trust bundles

Do not mirror private Root CA keys as part of application artifacts.

## Offline Trust Distribution

```mermaid
flowchart TD
  RootCert[Root CA certificate] --> Bundle[Trust bundle]
  Bundle --> Hosts[Host trust stores]
  Bundle --> Apps[Application trust stores]
  Bundle --> K8s[ConfigMaps or Secrets for workloads]
```

Distribute public trust bundles through controlled configuration management. Rotate trust bundles before issuing from a new Root during migration.

