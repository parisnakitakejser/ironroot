# Local Development

<div class="ironroot-doc-meta" markdown>
  <span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
  <span class="ironroot-badge ironroot-badge--status ironroot-badge--in-progress">Status: In Progress</span>
</div>

This guide walks contributors through building, installing, running, debugging, patching, and verifying IronRoot directly from a local Git checkout. It is designed to help you easily test and validate the complete development workflow locally before committing your changes.

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

Verify tools:

```bash
go version
git --version
just --version
sqlite3 --version
```

`just` is the project task runner for build, test, docs, container, and Helm workflows.

## Clone The Repository

```bash
git clone https://github.com/parisnakitakejser/ironroot.git
cd ironroot
```

This is the upstream IronRoot repository. Fork it first if you are contributing from your own GitHub account.

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
go build -o bin/ironroot-dev ./cmd/ironroot-dev
go build -o bin/irtop ./cmd/irtop
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
./bin/ironroot-dev --help
./bin/irtop --help
```

Option B: install into your user PATH:

```bash
mkdir -p ~/.local/bin
cp bin/ironroot-server ~/.local/bin/
cp bin/ironroot-admin ~/.local/bin/
cp bin/ironroot-client ~/.local/bin/
cp bin/ironroot-dev ~/.local/bin/
cp bin/irtop ~/.local/bin/
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
ironroot-dev --help
ironroot-dev dev-init --help
irtop --help
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

## Local Config And Data Directories

`ironroot-dev` is a contributor-only helper CLI. It is installed by `just install-local` and is not required in production. It exists to keep local developer workflows versioned, testable, and easier to extend than shell-only task runner logic.

Prepare the local workspace with:

```bash
ironroot-dev dev-init
```

`ironroot-dev dev-init` is self-contained. It does not need to read files from the IronRoot checkout at runtime. By default it creates `.localdev` under the directory where you run it. To target another base directory, pass `--base-dir`:

```bash
ironroot-dev dev-init --base-dir /path/to/workspace
```

For the complete `ironroot-dev` command reference, including `--dry-run`, `--verbose`, `--force`, and output path options, see [ironroot-dev](../api-cli/ironroot-dev.md).

The command creates a neutral local development workspace under the selected base directory. It does not create a demo DNS name or host-specific certificate directory. Those are created later when you enroll a client and request a certificate for a specific name.

Generated local tree:

```text
.localdev/
  config/
  data/
  pki/
    root/
    intermediate/
  certs/
  logs/
  tmp/
  .gitignore
  README.txt
```

Meaning:

- `.localdev/config`: generated local config files.
- `.localdev/data`: SQLite database.
- `.localdev/pki/root`: local Root CA material created by `ironroot-admin`.
- `.localdev/pki/intermediate`: local Intermediate CA material created by `ironroot-admin`.
- `.localdev/certs`: issued test certificates; DNS-specific directories are created later by `ironroot-client request-cert`.
- `.localdev/logs`: optional redirected logs; IronRoot logs to stdout by default.
- `.localdev/tmp`: temporary local development files.
- `.localdev/.gitignore`: keeps generated keys, certs, tokens, and databases out of Git.

## Create Local Development Config

`ironroot-dev dev-init` generates:

```text
.localdev/config/config.yaml
```

The file is generated from a local config template compiled into the `ironroot-dev` binary:

```text
ironroot-dev
```

During generation, IronRoot writes SQLite and PKI paths as absolute paths under the selected base directory. This makes the generated config usable even if you later run `ironroot-server`, `ironroot-admin`, or `ironroot-client` from another directory.

The generated local config uses:

- SQLite at `<base>/.localdev/data/ironroot.db`.
- PKI material under `<base>/.localdev/pki`.
- API listen address `localhost:8443`.
- Telemetry disabled by default.
- JSON logs to stdout.

The commands below assume you run them from the workspace base directory. If you run them from somewhere else on the machine, pass the generated absolute config path printed by `ironroot-dev dev-init` anywhere the guide uses `--config`:

```bash
ironroot-admin security-check --config /path/to/workspace/.localdev/config/config.yaml
```

Start `ironroot-server` after the local Root and Intermediate CA files exist. Before that point the config is valid, but certificate issuance cannot work because the configured PKI files have not been created yet.

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

If the server exits with `bind: address already in use`, another process is already listening on `localhost:8443`. Stop the existing process, or run this local server on another port and use the same port in every client command:

```bash
IRONROOT_SERVER_ADDRESS=localhost:9443 \
  ironroot-server --config .localdev/config/config.yaml
```

Then use:

```bash
irtop --server http://localhost:9443
ironroot-client enroll --server http://localhost:9443 --hostname demo.local --token <token>
ironroot-client request-cert --server http://localhost:9443 --enrollment-id <enrollment_id> --dns demo.local --out .localdev/certs/demo.local
```

## Monitor The Local Server

Use `irtop` for a read-only terminal view of the local API, CA status, certificates, enrollments, tokens, telemetry, and recent audit events.

The default local development config leaves `server.tls.cert_file` and `server.tls.key_file` empty, so the local server listens with plain HTTP on port `8443`. The URL scheme must match the server config:

```bash
irtop --server http://localhost:8443
```

For a one-shot status check without the interactive UI:

```bash
irtop --server http://localhost:8443 --output text
```

You can also use the local `irtop` config example:

```bash
irtop --config examples/irtop.local.yaml
```

If you accidentally use HTTPS against the local HTTP server:

```bash
irtop --server https://localhost:8443
```

`irtop` will explain that the server appears to be responding with HTTP and suggest the local command above. It will not silently downgrade the connection.

Production should use HTTPS:

```bash
irtop --server https://ironroot.example.com:8443 --ca-file ./root-ca.crt
```

Do not use `--insecure-skip-verify` unless you are intentionally debugging TLS trust.

See [irtop](../api-cli/irtop.md) for all flags and keyboard shortcuts.

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

The token is stored in the SQLite database from `.localdev/config/config.yaml`. The server you enroll against must be running with that same config. If an older `ironroot-server` is still listening on `localhost:8443` from another workspace, enrollment will return `401 Unauthorized` with `invalid bootstrap token` because that server is reading a different database.

Check which database contains the token:

```bash
ironroot-admin list-tokens --config .localdev/config/config.yaml
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

Verify the issued certificate chain:

```bash
openssl verify \
  -CAfile .localdev/pki/root/root-ca.crt \
  -untrusted .localdev/pki/intermediate/intermediate-ca.crt \
  .localdev/certs/demo.local/tls.crt
```

## Install Local Trust On A Linux Machine

After you request a test certificate, the most common question is which CA file should be installed as the trusted certificate.

Install the **Root CA public certificate** as the trust anchor:

```text
.localdev/pki/root/root-ca.crt
```

or the equivalent trust-bundle copy:

```text
.localdev/pki/root/trust-bundle/root-ca.crt
```

Do **not** install these files as system trust:

- `.localdev/pki/root/root-ca.key`: Root CA private key. Never copy this to a Linux machine for trust.
- `.localdev/pki/intermediate/intermediate-ca.key`: Intermediate private key. This belongs only on the IronRoot server.
- `.localdev/pki/intermediate/intermediate-ca.crt`: public Intermediate CA certificate. Services should present it in the chain, but normal OS trust should anchor at the Root CA.

The Intermediate CA certificate is still important. It is included in:

```text
.localdev/pki/intermediate/ca-chain.crt
.localdev/certs/demo.local/ca-chain.crt
.localdev/certs/demo.local/fullchain.crt
```

Use those chain files when configuring a service such as nginx, Caddy, or an application that needs to serve the leaf certificate together with the Intermediate. Trust stores should receive the Root CA certificate.

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
