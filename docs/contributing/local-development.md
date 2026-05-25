# Local Development

<div class="ironroot-doc-meta" markdown>
  <span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
  <span class="ironroot-badge ironroot-badge--status ironroot-badge--in-progress">Status: In Progress</span>
</div>

This section is the contributor workflow for running IronRoot from a local Git checkout. It is split into focused pages so you can jump directly to the task you need.

## Local Development Pages

| Page | Use it for |
| --- | --- |
| [Up And Running](local-development-up-and-running.md) | Build binaries, initialize `.localdev`, create local PKI, start the server, enroll a client, request a certificate, and monitor with `irtop`. |
| [Multi-Root CA Development](local-development-multi-root-ca.md) | Work with the multi-root and multi-intermediate CA metadata model, RBAC roles, token policies, example YAML, and `irtop` hierarchy view. |
| [RBAC Development](local-development-rbac.md) | Seed local RBAC role and token-policy metadata, verify `/v1/status/ca-hierarchy`, and test the `irtop` access display. |
| [Testing And Verification](local-development-testing.md) | Run unit tests, e2e tests, docs builds, lint/vet, telemetry checks, and the recommended patch verification loop. |
| [Troubleshooting](local-development-troubleshooting.md) | Fix common local problems such as wrong database/config, invalid bootstrap token, port conflicts, missing PKI files, TLS trust errors, and MkDocs dependencies. |

## Recommended First Path

For a fresh checkout, follow this order:

1. [Up And Running](local-development-up-and-running.md)
2. [Testing And Verification](local-development-testing.md)
3. [Troubleshooting](local-development-troubleshooting.md) only if something fails

If you are working on CA hierarchy or `irtop` visualization, also read [Multi-Root CA Development](local-development-multi-root-ca.md). If you are working on roles, permissions, or token policies, read [RBAC Development](local-development-rbac.md).

## Local Workspace Convention

Most contributor commands use a generated `.localdev` directory:

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
```

Generated private keys, certificates, SQLite databases, logs, and temporary files must stay out of Git.

## Important Local Assumptions

- The default local API listens on `localhost:8443`.
- The default local server uses HTTP, not HTTPS, because `server.tls.cert_file` and `server.tls.key_file` are empty.
- Bootstrap tokens are stored in the SQLite database from the config passed to `ironroot-admin create-token`.
- The server used for enrollment must be running with the same config/database that created the token.
- Local demo CA material is disposable. Do not reuse it in production.
