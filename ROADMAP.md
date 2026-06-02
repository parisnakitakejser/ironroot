# IronRoot Roadmap

This roadmap outlines the planned development and feature ideas for IronRoot. Features are grouped by code and component areas to make it easy to find areas to contribute to or track.

## Core Server & Binary Usage

Features focused on the central PKI server binary (`ironroot-server`), its architecture, and self-hosted environments.

### Planned
- **PostgreSQL backend support:** Enable clustering and high-availability deployments by supporting PostgreSQL alongside the default SQLite backend.
- **mTLS support:** Secure communication channels between the server, agents, and administration CLIs.
- **Admin API for token lifecycle:** Manage bootstrap and enrollment tokens programmatically via a dedicated, secure API.

### Research & Future Ideas
- **High Availability (HA) via RAFT database:** Research using a Raft-replicated database (such as rqlite or an integrated Raft library) for zero-dependency high-availability binary deployments without requiring a separate PostgreSQL instance.
- **SQLite replication/backup tooling:** Streamlined, zero-downtime backup mechanisms for standalone SQLite environments.
- **Memory-backed ephemeral mode:** Run the server on an in-memory database (`:memory:`) that loads from and saves to a signed backup file. This allows memory-only operations on high-security hosts where writing state to physical disks must be avoided.
- **ACME protocol support (RFC 8555):** Build a lightweight ACME challenge responder and directory endpoint to allow standard tools like `certbot` or ingress controllers to renew certificates automatically.
- **WASM-based validation plugins:** Support loading WebAssembly (WASM) binaries to execute custom, sandboxed policy validation checks on incoming certificate requests.
- **Multi-tenant CA partitioning:** Partition database and API structures to host independent teams and workloads securely on a single shared server instance.

---

## Cloud Native & Kubernetes

Features and improvements for deploying IronRoot inside Kubernetes clusters.

### Planned
- **High Availability (HA) deployments:** Support and document multi-replica HA deployments with Kubernetes anti-affinity, Pod Disruption Budgets (PDBs), and external database backend configuration.
- **cert-manager integration:** Build a custom issuer for cert-manager to automatically provision and rotate certificates for Kubernetes workloads.

### Research & Future Ideas
- **SPIFFE/SPIRE integration:** Support zero-trust identity provisioning by integrating with SPIFFE.
- **GitOps-native operations:** Native continuous-delivery workflows for managing CA policy configurations.
- **Multi-cluster trust distribution:** Automated distribution of intermediate certificates across multiple independent Kubernetes clusters.
- **Cross-namespace certificate replication:** A lightweight operator to mirror issued TLS secrets into other Kubernetes namespaces, simplifying trust setup for shared ingress proxies and service meshes.
- **Kubelet client certificate bootstrapping:** Accept and sign standard Kubernetes node CSRs (Certificate Signing Requests), allowing IronRoot to act as an external signer for secure node-to-master bootstrapping.

---

## Terminal UI & Operator Tooling (`irtop` & CLI)

Enhancements to the terminal console (`irtop`), administration tools (`ironroot-admin`), and client CLI (`ironroot-client`).

### Planned
- **Interactive CA lifecycle management:** Support promotion, rotation, and graceful retirement of intermediates directly from the CLI or console.
- **Lightweight client daemon:** A background daemon for automatic enrollment and certificate renewal on non-containerized hosts.

### Research & Future Ideas
- **Interactive live logs in `irtop`:** A real-time log-streaming panel inside the terminal UI filtered by tracing context.
- **Expanded diagnostics and check commands:** More extensive automated hygiene checks for validating permissions, configurations, and connectivity.
- **Break-glass emergency API freeze:** A quick-action command or `irtop` control to instantly freeze intermediate CA issuance or pause the server APIs during a suspected security incident.
- **Interactive CA trust tree visualizer:** A terminal UI view in `irtop` that renders a dynamic tree diagram of roots, intermediates, active workloads, and remaining lifetimes.

---

## Security, Cryptography & Compliance

Features addressing trust boundaries, cryptographic operations, and hardware integration.

### Planned
- **CRL (Certificate Revocation List) support:** Implement standard CRL generation and distribution points.
- **OCSP (Online Certificate Status Protocol) support:** Provide real-time revocation verification via an OCSP responder.
- **CA rollover workflows:** Automated, smooth transition from expiring intermediates to new ones.

### Research & Future Ideas
- **TPM & HSM integration:** Support hardware-backed private keys via PKCS#11 or TPM 2.0 to protect CAs.
- **SSH certificate authority support:** Manage and issue SSH host and user certificates alongside X.509.
- **Post-quantum cryptography (PQC) exploration:** Research and experiment with post-quantum signature algorithms for root and intermediate CAs.
- **Private Certificate Transparency (CT) log integration:** Log issued certificates to an append-only ledger so clients can verify Signed Certificate Timestamps (SCTs) and prevent rogue, untracked intermediate signing.
- **EST support (RFC 7030):** Support Enrollment over Secure Transport to let hardware routers, firewalls, and embedded devices bootstrap credentials natively.
- **Multi-operator FIDO2 approvals:** Require physical security keys (like YubiKeys) from multiple authorized administrators to sign off on high-risk CA operations or configuration changes.
- **CA Policy-as-Code (OPA/Rego integration):** Enforce declarative validation rules written in Rego or YAML, enabling automated verification of allowed domain constraints and certificate parameters in CI/CD before deployment.
- **Built-in SCEP responder (RFC 8894):** Implement standard Simple Certificate Enrollment Protocol support to automatically provision certificates for MDM platforms and managed mobile devices.
- **Shamir's Secret Sharing key unlocking:** Enable threshold-based software key splitting (e.g., needing 3 out of 5 shares to unlock the private key) as a resilient alternative to HSM operations.

---

## Ecosystem & Airgap Operations

Features for air-gapped environments, observability dashboards, and external packaging.

### Planned
- **Airgap package synchronization:** Streamlined, offline packaging and synchronization of container images, Helm charts, and binaries.
- **Standard dashboards:** Example Grafana dashboards pre-configured for OpenTelemetry metrics emitted by the server.

### Research & Future Ideas
- **Offline root CA bootstrap tools:** Enhanced guidance and utilities for generating trust roots on isolated, single-board computers or secure hardware enclaves.
- **Offline CRL/OCSP generation:** Tooling to build and sign revocation lists on air-gapped systems and securely transfer them to public networks.
- **Data diode unidirectional streaming:** Support forwarding telemetry, logs, and CRLs across a physical hardware data diode to a lower-security monitoring zone, with zero inbound paths back into the air-gapped core.
- **RAM-only offline ISO generator:** A utility to build an immutable, bootable Linux ISO configured to run entirely in memory on isolated laptops or single-board computers, creating a secure, clean-room environment for root signing operations.
