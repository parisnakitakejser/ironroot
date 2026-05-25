# AGENT.md

AI collaboration guide for IronRoot. Use this file to get oriented before changing code. Prefer the concrete patterns below over inventing new architecture.

## Project Overview

IronRoot is a Go-based PKI control plane for local, binary, container, Kubernetes, and air-gapped deployments. It provides:

- `ironroot-server`: HTTP/HTTPS API for enrollment, certificate issuance/renewal/revocation, status, audit, and metrics.
- `ironroot-admin`: operator CLI for local Root/Intermediate CA material, bootstrap tokens, security checks, and admin API actions.
- `ironroot-client`: host/client CLI for enrollment and certificate requests.
- `ironroot-dev`: contributor helper for `.localdev` workspace generation.
- `irtop`: Bubble Tea terminal dashboard for read-only operational monitoring.

Core architecture:

- Go 1.26 module: `github.com/parisnakitakejser/ironroot`.
- API: `go-chi/chi`, `net/http`, OpenTelemetry middleware, optional Prometheus `/metrics`.
- Storage: SQLite only, via `github.com/mattn/go-sqlite3`, with idempotent migrations in code.
- PKI: standard library `crypto/x509`; offline Root CA helpers and online Intermediate signing.
- CLIs: Cobra for admin/client/dev; standard `flag` for `irtop` and server.
- TUI: Bubble Tea + Lip Gloss.
- Docs: MkDocs Material under `docs/`, configured by `mkdocs.yml`.

Runtime assumptions:

- Local development often uses plain HTTP on `localhost:8443` because server TLS files are empty.
- Production/shared environments should use HTTPS and Root CA trust material.
- Server can start with missing CA material; it logs a warning and disables issuance via `ca.DisabledAuthority`.
- SQLite is serialized with `db.SetMaxOpenConns(1)`.

## Repository Structure

- `cmd/server`: server entrypoint. Loads config, configures telemetry/logging, opens DB, migrates, loads CA authority, starts HTTP server.
- `cmd/ironroot-admin`, `cmd/ironroot-client`, `cmd/ironroot-dev`: thin entrypoints into `internal/cli/...`.
- `cmd/irtop`: terminal monitor entrypoint and CLI flag handling.
- `internal/api`: chi router, HTTP handlers, JSON/Pem response helpers, status endpoints, API metrics middleware.
- `internal/audit`: audit logger wrapper over `db.Store`.
- `internal/ca`: CA authority interface, file-backed signer, local Root/Intermediate generation, certificate inspection/chain verification helpers.
- `internal/cli/admin`: Cobra admin command tree, local CA commands, token lifecycle, bootstrap/security checks, API-backed actions.
- `internal/cli/client`: Cobra client command tree, enrollment, CSR/key generation, certificate output writing.
- `internal/cli/dev`: `ironroot-dev dev-init` workspace/config scaffolding.
- `internal/config`: server config defaults, YAML/viper loading, `IRONROOT_*` env overrides, OTEL env normalization.
- `internal/crypto`: bootstrap token generation/hashing helpers.
- `internal/db`: `Store` interface, SQLite implementation, migrations, repository methods.
- `internal/enrollment`: bootstrap token validation and enrollment service.
- `internal/irtop`: TUI config/profile loading, API client, Bubble Tea model, render helpers, status types.
- `internal/securitycheck`: security check model, runner, checks, renderers, bootstrap guide.
- `internal/telemetry`: logger, metrics, trace helpers, OTLP/Prometheus setup.
- `pkg/client`: reusable IronRoot API client used by CLIs.
- `tests/e2e`: end-to-end tests.
- `examples`: local/server configs, `irtop.config`, Podman/Caddy/nginx/Python HTTPS, OTEL/Grafana examples.
- `deploy`: Kubernetes manifests and Helm chart.
- `docs`: MkDocs source.

Layer boundaries:

- API handlers depend on `db.Store`, `ca.Authority`, `audit.Logger`, and config via `api.Dependencies`.
- Business logic should live in packages like `internal/enrollment`, `internal/ca`, `internal/securitycheck`, not inside command entrypoints.
- Reusable HTTP client code belongs in `pkg/client`; TUI-specific client/rendering belongs in `internal/irtop`.
- Keep deployment assets and docs in sync with config and CLI changes.

## Coding Standards

- Use idiomatic Go, `gofmt`, small functions where practical, table tests for parsing/validation.
- File/package names are short lowercase: `handlers.go`, `repositories.go`, `localpki.go`, `config.go`.
- Public structs use exported fields when they cross package boundaries; DB models are simple structs in `internal/db`.
- Error handling: return errors with context using `fmt.Errorf("...: %w", err)` where wrapping is useful. HTTP handlers use `writeError`.
- CLI errors return from `RunE`; admin commands can use `ExitError` for controlled exit codes.
- Logging: server uses `slog` via `telemetry.NewLogger`; avoid ad hoc loggers.
- Context: pass `context.Context` through DB/API/CA/telemetry paths. Use spans around workflows and DB operations.
- Time: store and compare UTC timestamps; output RFC3339 in CLIs.
- Security-sensitive output: never print private key material, raw token hashes, or passwords.

## Existing Patterns AI Should Follow

- Dependency injection:
  - `api.NewRouter(api.Dependencies{...})`
  - `enrollment.Service{Store: ...}`
  - `securitycheck.DefaultRunner().Run(...)`
- DB:
  - Add methods to `db.Store` and `SQLStore` together.
  - Use repository methods in `internal/db/repositories.go`; add migration SQL to `internal/db/migrations.go`.
  - Record DB telemetry using `telemetry.RecordDatabase` where existing methods do.
- API:
  - Add routes in `internal/api/router.go`, implementation in `handlers.go` or `status.go`.
  - Use `decode`, `writeJSON`, `writeError`, `writePEM`.
  - Update status/TUI types if adding status fields.
- CLI:
  - Admin/client use Cobra and should keep usage quiet unless useful.
  - Client commands should use `tracedCommand` for telemetry/debug trace IDs.
  - Prefer existing output helpers and write files with deliberate permissions.
- PKI:
  - Use standard library `crypto/x509`.
  - Preserve Root CA offline assumptions and Intermediate online signing model.
  - Root CA defaults are production-minded; do not weaken without explicit request.
- Telemetry:
  - Reuse `telemetry.Configure`, `StartSpan`, `EndSpan`, `RecordCommand`, `SanitizeLabel`.
  - Avoid high-cardinality metric labels.
- Tests:
  - Local PKI tests must use temp dirs and generated material.
  - Prefer focused package tests plus `go test ./...` for cross-package changes.

Avoid duplicating:

- Token hashing/generation: use `internal/crypto`.
- API client calls: use `pkg/client` or `internal/irtop.Client`.
- Path expansion for `irtop`: use existing `expandPath`.
- Config defaults: use `config.Default()` for server, `irtop.DefaultConfig()` for TUI profile defaults.

## Terminal UI / UX Rules

`irtop` uses Bubble Tea model/update/view in `internal/irtop/model.go`.

State model:

- `stateLoading`: show loading screen immediately.
- `stateLoaded`: show selected view.
- `stateEmpty`: explicit no-data panel.
- `stateError`: startup/API error panel.

Profile behavior:

- Config file is `~/.ironroot/config`, YAML with top-level `profiles`.
- `default_profile` selects startup profile.
- `--profile` can override startup profile.
- `--server` can start a temporary `cli` profile without any config file.
- TUI switching happens in-process: `p` opens selector, `up/down` or `k/j` navigates, `enter` switches, `[` and `]` cycle.
- Switching profiles rebuilds the API client, resets snapshot, enters loading state, and refreshes.
- Header must show active profile.

Layout/style:

- Use existing Lip Gloss styles: `headerStyle`, `tabStyle`, `activeTabStyle`, `panelStyle`, `warnStyle`, `okStyle`, `badStyle`.
- Keep panels readable in narrow terminals; use `contentWidth`, `panelContentWidth`, `truncate`, `row`, and `metricLine`.
- Add keyboard help in `helpView` and footer when introducing controls.
- Never render an empty/broken screen during startup; loading or error must be visible.
- `RenderText` is for non-TUI output; keep it deterministic and simple.

## Configuration System

Server config (`internal/config`):

- Default path: `configs/server.yaml` unless `--config` or `IRONROOT_CONFIG` is used.
- Viper reads YAML and env vars with prefix `IRONROOT`; dots map to underscores.
- OTEL env vars are normalized: `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_EXPORTER_OTLP_PROTOCOL`, `OTEL_SERVICE_NAME`, `OTEL_RESOURCE_ATTRIBUTES`, sampler vars.
- Main sections: `server`, `database`, `pki`, `telemetry`, `log`.
- SQLite DSNs are used for local/dev; other DB drivers are not implemented.

Local dev config:

- Generated by `ironroot-dev dev-init`.
- `.localdev/config/config.yaml` uses absolute SQLite/PKI paths.
- API address defaults to `localhost:8443`; TLS cert/key are empty for local HTTP.

`irtop` config:

- Default path: `~/.ironroot/config`.
- Required YAML shape:

```yaml
default_profile: local
profiles:
  local:
    endpoint: http://localhost:8443
    refresh: 5s
    default_view: overview
    output: tui
  production:
    endpoint: https://ironroot.example.com:8443
    ca_file: ~/ironroot/root-ca.crt
    token: ""
    insecure_skip_verify: false
    refresh: 10s
    default_view: security
    output: tui
```

- `endpoint` is normalized to `server`; `server` is also accepted. Do not define both with different values.
- Duplicate profile names, empty names, missing fields, invalid shapes, invalid YAML, and unknown fields are errors.
- `irtop --server http://localhost:8443` works without a config file by creating a temporary `cli` profile.

## Development Workflow

Common commands:

```bash
just build-local
just install-local
just run-server
just smoke-cli
just fmt
just fmt-check
just vet
just test
just test-e2e
just docs-build
just docs-serve
go test ./...
```

Build outputs:

- Local binaries go to `bin/`.
- Cross-platform release builds go to `dist/`.
- Do not commit generated `.localdev`, `bin`, `dist`, `site`, DBs, keys, certs, coverage output, or temporary build binaries.

Docs:

- `just docs-build` runs MkDocs strict mode.
- Update `README.md`, `docs/api-cli/*`, and relevant operations/contributing docs when changing CLI behavior or config formats.

Containers/deploy:

- Container build uses `deploy/container/Containerfile` by default through `just container-build`.
- Helm chart lives in `deploy/helm/ironroot`; lint/template/package recipes exist in `justfile`.

## AI Agent Rules

- Inspect surrounding code before modifying architecture, especially API/store/config/TUI paths.
- Prefer extending existing interfaces and helpers over adding new packages.
- Keep patches small and focused; avoid broad refactors with feature changes.
- Preserve behavior unless the user explicitly asks for a breaking change.
- Update tests with code changes. Use focused tests first, then `go test ./...` for shared behavior.
- Update docs/examples whenever user-facing flags, config, file outputs, ports, or workflows change.
- Do not add dependencies unless the standard library or existing dependencies are clearly insufficient.
- Do not move Root CA private key handling into server/container/Kubernetes flows casually.
- Do not silently downgrade HTTPS to HTTP; show actionable errors.
- Do not print token secret values except when intentionally creating a new bootstrap token.
- Do not revert unrelated dirty worktree changes.

## Performance & Resource Expectations

- Server should remain lightweight: SQLite, in-process migrations, standard HTTP server.
- DB max open conns is intentionally `1` for SQLite; do not raise casually.
- API status endpoints should be cheap enough for `irtop` polling.
- `irtop` refresh defaults to `5s`; profile switches should refresh immediately.
- `irtop` snapshots require `/v1/status/overview`; other status calls are best-effort and currently ignored on failure.
- TUI rendering should be deterministic and avoid expensive work in `View`; network calls belong in commands (`tea.Cmd`).
- HTTP clients use timeouts; keep network error messages actionable.
- Avoid high-cardinality telemetry labels; sanitize labels.

## Common Pitfalls

- `ironroot-admin create-token --config X` writes to the SQLite DB in config `X`; enrollment must hit a server using the same DB/config.
- Port `8443` conflicts are common in local dev. Use another `IRONROOT_SERVER_ADDRESS` and matching client URL.
- Local server config is HTTP if TLS cert/key are empty. `irtop --server https://localhost:8443` will fail with an HTTP/HTTPS hint.
- `irtop` config requires top-level `profiles`; old top-level single-profile syntax is intentionally not supported.
- `default_profile` must match a profile name. `--profile` also must match.
- CA authority may be disabled at server startup if Intermediate files are missing; status can still work, issuance cannot.
- `security-check` can fail intentionally based on severity thresholds.
- Docs strict mode reports pages not in nav as INFO; broken links/config failures are the real blockers.
- SQLite schema migrations are code strings, not separate migration files.
- `site/` is generated docs output; avoid treating it as source.

