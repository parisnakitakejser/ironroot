# Airgap Overview

An air-gapped environment has no direct Internet access. Software, trust bundles, container images, Helm charts, and binaries move through controlled distribution paths.

IronRoot's model separates offline trust creation from online certificate operations:

```mermaid
flowchart LR
  Offline[Offline machine<br/>Root CA private key] -->|signed Intermediate cert| Transfer[approved transfer media]
  Transfer --> Online[Online IronRoot server<br/>Intermediate CA]
  Online --> Clients[clients and services]
  Offline -->|Root CA public cert| Trust[trust bundle distribution]
  Trust --> Clients
```

The online server never needs the Root CA private key.
