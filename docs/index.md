# IronRoot

**Airgap-first trust infrastructure**

IronRoot is a modern internal PKI platform for offline-root security, online issuing CAs, observable certificate operations, Kubernetes-native deployment, and self-hosted infrastructure.

## Why IronRoot exists

Many teams need internal TLS and machine identity without outsourcing trust, exposing root keys, or adopting a heavyweight enterprise PKI stack. IronRoot is built for platform engineers, homelab operators, Kubernetes users, and air-gapped environments that need automation with a clear trust boundary.

## The name

**Iron** means hardened infrastructure, durability, security, industrial-grade systems, and an airgap-first mindset.

**Root** means Root CA, trust anchor, certificate trust chain, and identity foundation.

Together, **IronRoot represents hardened trust infrastructure designed for modern self-hosted and air-gapped environments.**

Alternative taglines used across the project:

- Observable PKI for modern infrastructure
- Offline-root PKI for Kubernetes and homelabs
- Modern trust infrastructure for air-gapped environments

## Main features

- Offline Root CA with online Intermediate CA
- REST API, admin CLI, and client CLI
- Client-side private key generation
- SQLite by default with a storage interface ready for PostgreSQL
- OpenTelemetry tracing and metrics across CLI and server
- Podman container build and Kubernetes manifests
- Apache 2.0 open source project structure

## Architecture-first learning path

New operators should understand the trust model before installing IronRoot:

1. [Component Architecture](architecture/components.md)
2. [Offline Root CA](architecture/offline-root-ca.md)
3. [Online Intermediate CA](architecture/online-intermediate-ca.md)
4. [IronRoot API Server](architecture/api-server.md)
5. [Security Boundaries](architecture/trust-boundaries.md)
6. [Operations](operations/index.md)
