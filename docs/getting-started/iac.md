# 7. Infrastructure As Code

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status ironroot-badge--in-progress">Status: In Progress</span>
</div>

IronRoot configuration should be reviewable, repeatable, and deployable. Use YAML files as the source of truth for server settings and RBAC.

## Prerequisites

- Understand [RBAC And Security](rbac-security.md).
- Have a Git repository for environment configuration.

## Suggested Repository Layout

```text
infra/
  ironroot/
    dev/
      config.yaml
      rbac/
        roots.yaml
        intermediates.yaml
        roles.yaml
        bindings.yaml
        token-policies.yaml
    staging/
      config.yaml
      rbac/
        ...
    production/
      config.yaml
      rbac/
        ...
```

## Configure RBAC File Loading

```yaml
rbac:
  enabled: true
  mode: file
  paths:
    - config/rbac/*.yaml
    - config/rbac/*.yml
```

The loader expands globs, sorts matched files, validates every resource, and applies them deterministically at startup.

## Promotion Workflow

```mermaid
flowchart LR
  Dev[dev manifests] --> Review[Pull request review]
  Review --> CI[lint and tests]
  CI --> Staging[staging deployment]
  Staging --> Approval[security approval]
  Approval --> Prod[production deployment]
```

Recommended checks:

```bash
go test ./internal/rbac ./internal/config
just docs-build
git diff -- config/rbac
```

## Secrets And Git

Keep these out of Git:

- private CA keys.
- token secrets.
- database files.
- password files.
- generated certificates unless intentionally publishing trust bundles.

Keep these in Git:

- reviewed `config.yaml` templates.
- RBAC manifests.
- CA metadata manifests.
- deployment values.
- runbooks and diagrams.

## Expected Outcome

You can version RBAC and environment configuration without writing SQL manually.

## Validation

Restart a local server after changing `.localdev/config/rbac/*.yaml`. Startup should either succeed with the new manifests or fail with a clear validation error.

## Troubleshooting

| Symptom | Check |
| --- | --- |
| Files load in unexpected order | Filenames are sorted; prefix with numbers if order matters for review readability. |
| Production drift | Compare deployed config against Git and avoid manual database edits. |
| Secret leaked into config | Replace inline secrets with mounted files, environment injection, or secret manager references. |

## Next Step

Continue to [Production Deployment](production.md).
