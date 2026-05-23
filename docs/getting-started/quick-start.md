# Quick Start

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

Build the binaries:

```bash
make build
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
bin/ironroot-client enroll --server http://localhost:8443 --token <token>
bin/ironroot-client request-cert --server http://localhost:8443 --enrollment-id <id> --dns node-01.local --out certs
```

