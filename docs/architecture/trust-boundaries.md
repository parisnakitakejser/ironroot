# Security Boundaries

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

IronRoot is designed around explicit trust boundaries. Each boundary limits what an attacker can reach if one layer is compromised.

```mermaid
flowchart TD
  subgraph Offline["Offline trust boundary"]
    Root[Root CA private key]
  end
  subgraph Online["Online issuing boundary"]
    Intermediate[Intermediate CA key]
    API[IronRoot API]
    DB[(Database)]
  end
  subgraph Operators["Operator boundary"]
    Admin[Admin CLI]
  end
  subgraph Workloads["Client/workload boundary"]
    Client[Client CLI]
    Key[Workload private key]
  end
  subgraph Platform["Platform boundary"]
    K8s[Kubernetes / Podman / host runtime]
    OTel[OpenTelemetry stack]
  end
  Root --> Intermediate
  Admin --> API
  Client --> API
  API --> DB
  API --> OTel
```

## Attack Surface Overview

```mermaid
flowchart LR
  Internet[Untrusted network] -. avoid direct exposure .-> API[API server]
  Operator[Trusted operator] --> Admin[Admin CLI]
  Workload[Workload host] --> Client[Client CLI]
  API --> Intermediate[Intermediate key]
  API --> DB[(SQLite / future PostgreSQL)]
  Runtime[Container or Kubernetes runtime] --> Secrets[Mounted CA Secret]
```

Primary controls:

- keep Root CA offline
- keep Intermediate CA permissions strict
- enable TLS before remote API exposure
- scope Kubernetes Secret access tightly
- use NetworkPolicy where possible
- audit important actions
- monitor with OpenTelemetry without leaking secrets

## Secure Deployment Model

The secure model is not just "install the server." It is:

1. Generate and store Root CA offline.
2. Sign an Intermediate CA with the Root CA.
3. Mount only Intermediate CA material and public Root/chain files into IronRoot.
4. Run the API with TLS and restricted filesystem permissions.
5. Use short-lived bootstrap tokens.
6. Let clients generate private keys locally and send CSRs only.
7. Back up the database and Intermediate CA material together.
8. Keep Root CA recovery material offline and encrypted.
