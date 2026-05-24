# Quick Start

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

Build the binaries:

```bash
just build
```

Initialize the SQLite database:

```bash
IRONROOT_CONFIG=configs/server.yaml bin/ironroot-admin init-server
```

Create a bootstrap token:

```bash
IRONROOT_CONFIG=configs/server.yaml bin/ironroot-admin create-token --host node-01 --ttl 1h
```

Start the server:

```bash
IRONROOT_CONFIG=configs/server.yaml bin/ironroot-server
```

Enroll a client and request a certificate:

```bash
bin/ironroot-client enroll --server http://localhost:8443 --hostname node-01 --token <token>
bin/ironroot-client request-cert --server http://localhost:8443 --enrollment-id <id> --dns node-01.local --out certs
```
