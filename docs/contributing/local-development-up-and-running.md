# Local Development: Up And Running

<div class="ironroot-doc-meta" markdown>
  <span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
  <span class="ironroot-badge ironroot-badge--status ironroot-badge--in-progress">Status: In Progress</span>
</div>

This page gets a contributor checkout running with a local SQLite database, local Root CA, local Intermediate CA, server, client enrollment, certificate issuance, and `irtop`.

## Prerequisites

Required:

- Go
- Git
- `just`
- SQLite

Optional:

- Podman for container examples
- Helm for chart work
- Python and MkDocs dependencies for docs
- `golangci-lint` and `govulncheck` for deeper checks

Verify tools:

```bash
go version
git --version
just --version
sqlite3 --version
```

## Clone And Build

```bash
git clone https://github.com/parisnakitakejser/ironroot.git
cd ironroot
just build-local
```

Expected output:

```text
bin/
  ironroot-server
  ironroot-admin
  ironroot-client
  ironroot-dev
  irtop
```

Install into `~/.local/bin`:

```bash
just install-local
export PATH="$HOME/.local/bin:$PATH"
```

Verify:

```bash
ironroot-server --version
ironroot-admin --help
ironroot-client --help
ironroot-dev dev-init --help
irtop --help
```

## Initialize `.localdev`

```bash
ironroot-dev dev-init
```

This creates:

```text
.localdev/
  config/config.yaml
  config/rbac/local-rbac.yaml
  data/
  pki/root/
  pki/intermediate/
  certs/
  logs/
  tmp/
```

The generated config uses absolute paths under `.localdev`, SQLite at `.localdev/data/ironroot.db`, API address `localhost:8443`, file-based RBAC manifests under `.localdev/config/rbac`, telemetry disabled by default, and JSON logs to stdout.

## Create Local PKI

Generate a local Root CA:

```bash
ironroot-admin ca create-root \
  --name "IronRoot Local Root CA" \
  --key-password ironroot-local-root \
  --out .localdev/pki/root
```

Generate a local Intermediate CA:

```bash
ironroot-admin ca create-intermediate \
  --root-cert .localdev/pki/root/root-ca.crt \
  --root-key .localdev/pki/root/root-ca.key \
  --root-password ironroot-local-root \
  --password ironroot-local-intermediate \
  --out .localdev/pki/intermediate
```

Inspect the generated material:

```bash
ironroot-admin ca inspect \
  --output table \
  .localdev/pki/root/root-ca.crt \
  .localdev/pki/intermediate/intermediate-ca.crt
```

Run bootstrap checks:

```bash
ironroot-admin bootstrap \
  --config .localdev/config/config.yaml \
  --non-interactive \
  --acknowledge-risk
```

## Start The Server

```bash
ironroot-server --config .localdev/config/config.yaml
```

For quick iteration:

```bash
go run ./cmd/server --config .localdev/config/config.yaml
```

Or:

```bash
just run-server
```

The local server uses HTTP on `localhost:8443`.

## Create A Bootstrap Token

In another terminal:

```bash
ironroot-admin create-token \
  --config .localdev/config/config.yaml \
  --host demo.local \
  --ttl 24h
```

Copy the returned token. The token is stored in the SQLite database from `.localdev/config/config.yaml`.

## Enroll A Client

```bash
ironroot-client enroll \
  --server http://localhost:8443 \
  --hostname demo.local \
  --token <token>
```

`--hostname` must match the `--host` value used when the token was created. Copy the returned `enrollment_id`; certificate requests use that UUID, not the bootstrap token.

## Request A Test Certificate

```bash
ironroot-client request-cert \
  --server http://localhost:8443 \
  --enrollment-id <enrollment_id> \
  --dns demo.local \
  --out .localdev/certs/demo.local
```

Inspect generated files:

```bash
find .localdev/certs/demo.local -maxdepth 1 -type f -print
cat .localdev/certs/demo.local/README.txt
cat .localdev/certs/demo.local/metadata.json
```

Verify the issued certificate chain:

```bash
openssl verify \
  -CAfile .localdev/pki/root/root-ca.crt \
  -untrusted .localdev/pki/intermediate/intermediate-ca.crt \
  .localdev/certs/demo.local/tls.crt
```

## Monitor With irtop

```bash
irtop --server http://localhost:8443
```

One-shot text output:

```bash
irtop --server http://localhost:8443 --output text
```

Using the example config:

```bash
irtop --config examples/irtop.local.yaml
```

If you use HTTPS against the local HTTP server, `irtop` returns a clear HTTP/HTTPS mismatch hint. It does not silently downgrade.

## Local Trust

Install the Root CA public certificate as the trust anchor:

```text
.localdev/pki/root/root-ca.crt
```

Do not install private keys as trust material:

- `.localdev/pki/root/root-ca.key`
- `.localdev/pki/intermediate/intermediate-ca.key`

Debian/Ubuntu:

```bash
sudo cp .localdev/pki/root/root-ca.crt /usr/local/share/ca-certificates/ironroot-local.crt
sudo update-ca-certificates
```

Fedora/RHEL:

```bash
sudo cp .localdev/pki/root/root-ca.crt /etc/pki/ca-trust/source/anchors/ironroot-local.crt
sudo update-ca-trust
```
