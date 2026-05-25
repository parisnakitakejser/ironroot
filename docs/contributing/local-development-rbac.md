# Local Development: RBAC

<div class="ironroot-doc-meta" markdown>
  <span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
  <span class="ironroot-badge ironroot-badge--status ironroot-badge--in-progress">Status: In Progress</span>
</div>

Use this page when developing or testing Kubernetes-style RBAC manifests for multi-root and multi-intermediate CA flows.

## Current Boundary

RBAC is file-based and declarative. Developers should not write SQL to set up RBAC.

The server can load YAML manifests at startup and apply them into internal SQLite metadata tables. The database remains an implementation detail; Git-versioned YAML files are the source of truth.

The live certificate signing path still uses the current single active `ca.Authority`. RBAC enforcement before signing is future write-path work. Do not assume enrollment alone grants signing permission when implementing that future work.

## Configure RBAC Loading

RBAC is configured in the main IronRoot `config.yaml`:

```yaml
rbac:
  enabled: true
  mode: file
  paths:
    - config/rbac/*.yaml
    - config/rbac/*.yml
```

`ironroot-dev dev-init` generates:

```text
.localdev/config/rbac/local-rbac.yaml
```

and writes absolute RBAC glob paths into:

```text
.localdev/config/config.yaml
```

For repo-level examples, see:

```text
config/rbac/local-rbac.yaml
examples/ca-hierarchy.yaml
```

## Manifest Model

RBAC manifests use Kubernetes-style resources:

```yaml
apiVersion: ironroot.io/v1alpha1
kind: CARole
metadata:
  name: local-platform-issuer
spec:
  rules:
    - resources:
        - certificates
      verbs:
        - request
        - renew
        - view
      intermediateRef: local-intermediate
---
apiVersion: ironroot.io/v1alpha1
kind: CARoleBinding
metadata:
  name: local-platform-issuer-binding
spec:
  roleRef:
    kind: CARole
    name: local-platform-issuer
  subjects:
    - kind: Group
      name: platform
```

Supported local resource kinds:

- `User`
- `Group`
- `ServiceAccount`
- `RootCA`
- `IntermediateCA`
- `Role`, `CARole`, `GlobalRole`
- `RoleBinding`, `CARoleBinding`, `GlobalRoleBinding`
- `TokenPolicy`

Certificate-related permissions should use verbs such as:

- `request`
- `renew`
- `revoke`
- `view`
- `manage-intermediate`
- `manage-root`
- `request-token`

## Loading Rules

The loader:

- expands every configured glob
- ignores directories
- sorts matched files for deterministic load order
- supports multiple YAML documents per file
- validates required fields before applying anything
- rejects unknown fields
- returns clear startup errors for invalid manifests

RBAC is loaded at server startup. Restart the server after changing manifests.

## Test Locally

Follow [Up And Running](local-development-up-and-running.md), then start the server:

```bash
ironroot-server --config .localdev/config/config.yaml
```

On startup, the server loads every file matching the configured RBAC paths. If a manifest is invalid, startup fails with a `load RBAC manifests` error.

Verify the hierarchy endpoint:

```bash
curl -s http://localhost:8443/v1/status/ca-hierarchy | jq .
```

The response should include:

- `summary.roles`
- `summary.token_policies`
- `roots[].intermediates[].roles`
- `roots[].intermediates[].token_policies`

Verify `irtop`:

```bash
irtop --server http://localhost:8443
```

Press:

```text
5
```

The CA Health view should show an access line similar to:

```text
access policies=1 roles=1
```

## GitOps Workflow

Keep RBAC manifests in Git:

```text
config/
  rbac/
    roots.yaml
    intermediates.yaml
    roles.yaml
    bindings.yaml
    token-policies.yaml
```

Review RBAC changes like application code:

```bash
git diff -- config/rbac
go test ./internal/rbac ./internal/db ./internal/api ./internal/irtop
just docs-build
```

Production can use the same file model with reviewed manifests, CI validation, and deployment automation. The difference is operational discipline: production RBAC files should be reviewed, signed off, deployed through the release process, and paired with audit monitoring.

## Negative Local Checks

Break a role binding reference:

```yaml
roleRef:
  kind: CARole
  name: does-not-exist
```

Restart the server. It should fail during startup with a clear missing role error.

Break a token policy by removing `intermediateRef`:

```yaml
kind: TokenPolicy
metadata:
  name: invalid
spec:
  certificateTypes:
    - server
```

Restart the server. It should fail validation before applying the policy.

## Tests To Run

Focused tests:

```bash
go test ./internal/rbac ./internal/db ./internal/api ./internal/irtop ./cmd/irtop
```

Full validation:

```bash
go test ./...
just fmt-check
just vet
just docs-build
```

## Future Enforcement Checklist

When implementing signing-path RBAC, add tests that prove:

- a request without issuer permission is denied
- a token cannot request certificates from a different Intermediate CA
- requested DNS/SANs must match the policy
- requested TTL cannot exceed the policy maximum
- issuance limits are enforced
- renewal is denied when the policy disables renewal
- approval-required policies do not sign until approved
- every allow/deny decision writes an audit event

Keep enforcement close to the certificate request path, before `ca.Authority.SignCSR`.
