# Local Development: Testing And Verification

<div class="ironroot-doc-meta" markdown>
  <span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
  <span class="ironroot-badge ironroot-badge--status ironroot-badge--in-progress">Status: In Progress</span>
</div>

Use this page after you have the local environment running or when validating a patch.

## Fast Feedback

Run all unit tests:

```bash
just test
go test ./...
```

Run focused packages:

```bash
go test ./internal/irtop ./cmd/irtop
go test ./internal/api ./internal/db
go test ./internal/cli/admin ./internal/cli/client
```

Run e2e tests:

```bash
just test-e2e
```

## Formatting, Vet, Lint

Format code:

```bash
just fmt
```

Check formatting without changing files:

```bash
just fmt-check
```

Run vet:

```bash
just vet
```

Run lint if installed:

```bash
just lint
```

## Docs

Preview docs locally:

```bash
just docs-serve
```

Build docs in strict mode:

```bash
just docs-build
```

MkDocs may print informational messages for pages not included in navigation. Treat broken links, config errors, and build failures as blockers.

## Telemetry Testing

Start the local observability stack from `examples/otel/`, then run IronRoot with OTEL variables:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
OTEL_SERVICE_NAME=ironroot-local \
go run ./cmd/server --config .localdev/config/config.yaml
```

Run client commands in another terminal to generate traces and metrics.

## Recommended Patch Loop

```bash
git checkout -b fix/my-change
just fmt
just test
just test-e2e
just lint
just docs-build
just build-local
```

Then manually verify the local flow that your patch changed.

## High-Risk Areas

Run broader checks when touching:

- `internal/db`: migrations and repository tests
- `internal/api`: router and status tests
- `internal/ca`: local PKI tests and certificate chain verification
- `internal/cli/*`: command tests and e2e tests
- `internal/irtop`: package tests plus manual `irtop --server http://localhost:8443`
- docs or examples: `just docs-build`

## Manual Smoke Flow

After large changes:

```bash
just build-local
ironroot-dev dev-init --force
ironroot-admin ca create-root --name "IronRoot Local Root CA" --key-password ironroot-local-root --out .localdev/pki/root
ironroot-admin ca create-intermediate --root-cert .localdev/pki/root/root-ca.crt --root-key .localdev/pki/root/root-ca.key --root-password ironroot-local-root --password ironroot-local-intermediate --out .localdev/pki/intermediate
ironroot-admin bootstrap --config .localdev/config/config.yaml --non-interactive --acknowledge-risk
ironroot-server --config .localdev/config/config.yaml
```

In another terminal, create a token, enroll, request a cert, and open `irtop`.
