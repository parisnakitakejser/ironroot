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

---

## Terminal UI & Operator Tooling (`irtop` & CLI)

Enhancements to the terminal console (`irtop`), administration tools (`ironroot-admin`), and client CLI (`ironroot-client`).

### Planned
- **Interactive CA lifecycle management:** Support promotion, rotation, and graceful retirement of intermediates directly from the CLI or console.
- **Lightweight client daemon:** A background daemon for automatic enrollment and certificate renewal on non-containerized hosts.

### Research & Future Ideas
- **Interactive live logs in `irtop`:** A real-time log-streaming panel inside the terminal UI filtered by tracing context.
- **Expanded diagnostics and check commands:** More extensive automated hygiene checks for validating permissions, configurations, and connectivity.

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

---

## Ecosystem & Airgap Operations

Features for air-gapped environments, observability dashboards, and external packaging.

### Planned
- **Airgap package synchronization:** Streamlined, offline packaging and synchronization of container images, Helm charts, and binaries.
- **Standard dashboards:** Example Grafana dashboards pre-configured for OpenTelemetry metrics emitted by the server.

### Research & Future Ideas
- **Offline root CA bootstrap tools:** Enhanced guidance and utilities for generating trust roots on isolated, single-board computers or secure hardware enclaves.
- **Offline CRL/OCSP generation:** Tooling to build and sign revocation lists on air-gapped systems and securely transfer them to public networks.
