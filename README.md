# IronRoot

![Apache 2.0](https://img.shields.io/badge/license-Apache%202.0-blue)
![Go](https://img.shields.io/badge/go-1.26.3-00ADD8)
![CI](https://github.com/ironroot/ironroot/actions/workflows/ci.yaml/badge.svg)
![Docs](https://img.shields.io/badge/docs-MkDocs-526CFE)

> Airgap-first trust infrastructure

<!-- Logo placeholder: docs/assets/logo-placeholder.png -->

IronRoot is a modern internal PKI platform for air-gapped environments, offline-root security, observable certificate operations, Kubernetes-native deployments, and self-hosted infrastructure.

## What the name means

**Iron** represents hardened infrastructure, durability, security, industrial-grade systems, and an airgap-first mindset.

**Root** represents the Root CA, trust anchor, certificate trust chain, and identity foundation.

Together: **IronRoot represents hardened trust infrastructure designed for modern self-hosted and air-gapped environments.**

## Architecture

```text
Offline Root CA
    |
    | signs Intermediate CA
    v
Online Go PKI Server
    |
    | REST API + OpenTelemetry
    v
Admin CLI + Client CLI
    |
    | enroll / request / renew
    v
Homelab servers and Kubernetes services
```

## Quick start

```bash
make build
IRONROOT_CONFIG=configs/server.yaml bin/ironroot-admin init-server
IRONROOT_CONFIG=configs/server.yaml bin/ironroot-admin create-token --host node-01 --ttl 1h
IRONROOT_CONFIG=configs/server.yaml bin/ironroot-server
```

```bash
bin/ironroot-client enroll --server http://localhost:8443 --token <token>
bin/ironroot-client request-cert --server http://localhost:8443 --enrollment-id <id> --dns node-01.local --out certs
```

## OpenTelemetry

IronRoot instruments server endpoints and client commands. CLI spans propagate W3C Trace Context to the REST API so traces show enrollment, CSR validation, signing, metadata storage, and response flow.

## Kubernetes and Podman

Kubernetes manifests live in `deploy/kubernetes`. The Podman-compatible image build lives in `deploy/container/Containerfile`.

```bash
make container-build
kubectl apply -k deploy/kubernetes
```

## Documentation

Documentation is built with MkDocs Material:

```bash
make docs-serve
make docs-build
```

## Project links

- Documentation: `docs/`
- Roadmap: `ROADMAP.md`
- Contributing: `CONTRIBUTING.md`
- Security disclosure: `SECURITY.md`
- License: `LICENSE`

## License

IronRoot is licensed under the Apache License 2.0.

