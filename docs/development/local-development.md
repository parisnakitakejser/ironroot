# Local Development

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

This guide is the contributor path for building, installing, running, debugging, patching, and verifying IronRoot from a Git checkout.

## Prerequisites

Required:

- Go
- Git
- `just`
- SQLite

Optional but recommended:

- Podman for container testing
- Helm for chart work
- Python and MkDocs dependencies for docs
- `golangci-lint` for local linting
- `govulncheck` for vulnerability checks
- Make if you need the CI-compatible fallback tasks

Verify tools:

```bash
go version
git --version
just --version
sqlite3 --version
```

`just` is the preferred local task runner. The repository keeps a Makefile as a compatibility layer for CI and contributors who have not installed `just` yet.

## Clone The Repository

```bash
git clone https://github.com/OWNER/ironroot.git
cd ironroot
```

Replace `OWNER` with your GitHub user or organization.

## Build Binaries Locally

Use `just build-local` for the fast local development build. It only builds the binaries for your current operating system and CPU architecture.

```bash
just build-local
```

Equivalent direct Go commands:

```bash
go build -o bin/ironroot-server ./cmd/server
go build -o bin/ironroot-admin ./cmd/ironroot-admin
go build -o bin/ironroot-client ./cmd/ironroot-client
```

Expected output:

```text
bin/
  ironroot-server
  ironroot-admin
  ironroot-client
```

`just build` is kept as a compatibility alias for `just build-local`.

Only use cross-compilation when you need to validate release targets:

```bash
just build-linux
just build-macos
just build-all
```

## Install Local Development Binaries

Option A: run from `./bin`:

```bash
./bin/ironroot-server --version
./bin/ironroot-admin --help
./bin/ironroot-client --help
```

Option B: install into your user PATH:

```bash
mkdir -p ~/.local/bin
cp bin/ironroot-server ~/.local/bin/
cp bin/ironroot-admin ~/.local/bin/
cp bin/ironroot-client ~/.local/bin/
export PATH="$HOME/.local/bin:$PATH"
```

Or use the task runner:

```bash
just install-local
```

`just install-local` depends on `just build-local`, so it always rebuilds the current-platform binaries before installing them. Use it when you want the binaries in `~/.local/bin` to reflect the latest checkout.

Verify:

```bash
ironroot-server --version
ironroot-admin --help
ironroot-client --help
```

Linux bash PATH:

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

macOS zsh PATH:

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

## Go Install Workflow

`go install` installs binaries named after the command directories. IronRoot uses:

```bash
go install ./cmd/server
go install ./cmd/ironroot-admin
go install ./cmd/ironroot-client
```

This produces `server`, `ironroot-admin`, and `ironroot-client`. Use `just build-local` or `just install-local` when you need the server binary named `ironroot-server`.

## Local Config And Data Directories

Recommended local tree:

```text
.localdev/
  config/
  data/
  pki/
  certs/
  logs/
```

Create it:

```bash
mkdir -p .localdev/config .localdev/data .localdev/pki .localdev/certs .localdev/logs
```

Meaning:

- `.localdev/config`: local config files.
- `.localdev/data`: SQLite database.
- `.localdev/pki`: Root and Intermediate CA material for local testing.
- `.localdev/certs`: issued test certificates.
- `.localdev/logs`: optional redirected logs; IronRoot logs to stdout by default.

## Create Local Development Config

```bash
cp examples/config.local.yaml .localdev/config/config.yaml
```

Or initialize everything:

```bash
just dev-init
```

The local config uses:

- SQLite at `.localdev/data/ironroot.db`.
- PKI material under `.localdev/pki`.
- API listen address `localhost:8443`.
- Telemetry disabled by default.
- JSON logs to stdout.

## Local PKI Bootstrap Flow

Generate a local Root CA:

```bash
ironroot-admin ca create-root \
  --name "IronRoot Local Root CA" \
  --key-password ironroot-local-root \
  --out .localdev/pki/root
```

The local command uses the same Root CA defaults that production operators start from: ECDSA P-384, encrypted private key, 20 year validity, max path length 1, and CA signing only. The generated `root-ca.key` is sensitive; the generated `root-ca.crt` and `trust-bundle/root-ca.crt` are public trust material.

Generate a local Intermediate CA:

```bash
ironroot-admin ca create-intermediate \
  --root-cert .localdev/pki/root/root-ca.crt \
  --root-key .localdev/pki/root/root-ca.key \
  --root-password ironroot-local-root \
  --password ironroot-local-intermediate \
  --out .localdev/pki/intermediate
```

Inspect the generated CA material:

```bash
ironroot-admin ca inspect \
  --output table \
  .localdev/pki/root/root-ca.crt \
  .localdev/pki/intermediate/intermediate-ca.crt
```

Run the first-run bootstrap guide:

```bash
ironroot-admin bootstrap \
  --config .localdev/config/config.yaml \
  --non-interactive \
  --acknowledge-risk
```

## Run Server Locally

Run the built binary:

```bash
ironroot-server --config .localdev/config/config.yaml
```

Run with `go run` for quick iteration:

```bash
go run ./cmd/server --config .localdev/config/config.yaml
```

Or use:

```bash
just run-server
```

## Use Admin CLI Locally

```bash
ironroot-admin security-check --config .localdev/config/config.yaml
```

Create a bootstrap token:

```bash
ironroot-admin create-token \
  --config .localdev/config/config.yaml \
  --host demo.local \
  --ttl 24h
```

## Use Client CLI Locally

Enroll:

```bash
ironroot-client enroll \
  --server http://localhost:8443 \
  --hostname demo.local \
  --token <token>
```

`--hostname` must match the `ironroot-admin create-token --host` value. Without it, the client sends the machine's OS hostname, which is useful for real hosts but confusing in local demos.

Copy the returned `enrollment_id`. Certificate requests use that UUID, not the bootstrap token.

Request a test certificate:

```bash
ironroot-client request-cert \
  --server http://localhost:8443 \
  --enrollment-id <enrollment_id> \
  --dns demo.local \
  --out .localdev/certs/demo.local
```

The output directory is created automatically. Existing files are protected; use `--overwrite` only when you intentionally want to replace `tls.key`, `tls.crt`, `fullchain.crt`, metadata, and fingerprints.

Inspect generated files:

```bash
find .localdev/certs/demo.local -maxdepth 1 -type f -print
cat .localdev/certs/demo.local/README.txt
cat .localdev/certs/demo.local/metadata.json
```

## Debugging IronRoot Locally

Run with debug logs:

```bash
IRONROOT_LOG_LEVEL=debug go run ./cmd/server --config .localdev/config/config.yaml
```

Use health endpoints:

```bash
curl http://localhost:8443/healthz
curl http://localhost:8443/readyz
```

Inspect SQLite:

```bash
sqlite3 .localdev/data/ironroot.db ".tables"
```

Inspect a certificate with OpenSSL as an optional debugging tool:

```bash
openssl x509 -in .localdev/certs/demo.local/tls.crt -text -noout
```

Delve can be used if installed:

```bash
dlv debug ./cmd/server -- --config .localdev/config/config.yaml
```

## Patch And Verify A Fix

Recommended loop:

```bash
git checkout -b fix/my-change
just fmt
just test
just test-e2e
just lint
just docs-build
just build-local
```

Then manually verify the local flow you changed before opening a pull request.

## Testing

Run all unit tests:

```bash
just test
go test ./...
```

Run one package:

```bash
go test ./internal/...
```

Run e2e tests:

```bash
just test-e2e
```

Tests live beside internal packages and under `tests/e2e`.

## Docs Local Development

Preview docs:

```bash
just docs-serve
```

Build docs:

```bash
just docs-build
```

Docs are built from `main` and published as generated static files to the `website` branch.

## Local Telemetry Testing

Start the local observability stack from `examples/otel/`, then run IronRoot with OTEL variables:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
OTEL_SERVICE_NAME=ironroot-local \
go run ./cmd/server --config .localdev/config/config.yaml
```

Run client commands in another terminal to generate traces and metrics.

## Troubleshooting

Binary not found:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

Port `8443` already in use:

```bash
lsof -i :8443
```

SQLite permission problems:

```bash
ls -ld .localdev/data
```

TLS trust errors:

- Confirm the Root CA is installed in the OS or browser trust store.
- Confirm the certificate DNS name matches the URL.

Missing PKI files:

```bash
find .localdev/pki -type f -maxdepth 3
```

Bootstrap not completed:

```bash
ironroot-admin bootstrap --config .localdev/config/config.yaml --non-interactive --acknowledge-risk
```

Client cannot connect:

```bash
curl http://localhost:8443/healthz
```

Wrong config path:

```bash
ironroot-server --config .localdev/config/config.yaml
```

MkDocs missing dependencies:

```bash
just docs-install
```

Clean local state:

```bash
just dev-clean
```
