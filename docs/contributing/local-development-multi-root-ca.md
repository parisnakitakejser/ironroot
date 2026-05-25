# Local Development: Multi-Root CA

<div class="ironroot-doc-meta" markdown>
  <span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
  <span class="ironroot-badge ironroot-badge--status ironroot-badge--in-progress">Status: In Progress</span>
</div>

Use this page when developing or validating the multi-root and multi-intermediate CA model, RBAC metadata, token policies, status APIs, and `irtop` visualization.

## Current Local Boundary

The local signing path is still backward compatible with one active online `ca.Authority`. The multi-root work currently adds the scalable metadata and read-only monitoring surface:

- Root CA metadata
- Intermediate CA metadata
- RBAC role bindings
- token/request policy metadata
- certificate counts per Intermediate CA
- `/v1/status/ca-hierarchy`
- `irtop` CA Health hierarchy rendering

Issuer import, issuer selection, approval mutation APIs, and request-scoped token enforcement are future write-path work.

## Example YAML

The reference structure lives in:

```text
examples/ca-hierarchy.yaml
```

It models:

- production, staging, and lab Root CAs
- multiple Intermediate CAs under a Root CA
- team and namespace ownership
- certificate signing restrictions
- max TTL and issuance limits
- renewal permissions
- approval requirements
- RBAC roles
- token policies

Inspect it with:

```bash
cat examples/ca-hierarchy.yaml
```

## Status API

After the server is running, query the hierarchy status:

```bash
curl http://localhost:8443/v1/status/ca-hierarchy
```

The endpoint is monitoring-only. It returns hierarchy metadata and never returns private Root keys, Intermediate keys, token secrets, or token hashes.

If no multi-root metadata exists yet, IronRoot falls back to existing `ca_config` rows where possible and marks the response as a legacy projection.

## irtop Visualization

Run:

```bash
irtop --server http://localhost:8443
```

Open the CA Health view:

```text
5
```

The CA view displays:

- Root CAs
- Intermediate CAs
- environment labels
- owner/namespace metadata
- active/revoked certificate counts
- max TTL
- renewal and approval state
- token policy counts
- RBAC role counts
- warnings and legacy fallback state

## Development Checklist

When changing this area, check all of these packages:

```bash
go test ./internal/db ./internal/api ./internal/irtop ./cmd/irtop
```

Then run the full baseline:

```bash
go test ./...
just fmt-check
just vet
just docs-build
```

For role and token-policy metadata testing, use [RBAC Development](local-development-rbac.md).

## Architecture References

- [Multi-Root CA Architecture](../architecture/multi-root-ca.md)
- [Trust Model](../architecture/trust-model.md)
- [Online Intermediate CA](../architecture/online-intermediate-ca.md)
- [irtop Observability](../observability/irtop.md)

## Implementation Notes

Keep this model modular:

- DB metadata belongs in `internal/db`.
- Status projection belongs in `internal/api/status.go`.
- Read-only TUI display belongs in `internal/irtop`.
- Signing enforcement should happen before `ca.Authority.SignCSR`.
- Authorization must be request-scoped and auditable.

Do not add shortcuts that grant signing permission from enrollment alone. Enrollment proves host identity for the local flow; it is not an issuer authorization model.
