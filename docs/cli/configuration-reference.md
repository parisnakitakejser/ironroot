# Configuration Reference

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

IronRoot configuration includes server listener settings, database settings, PKI paths, file-based RBAC settings, telemetry settings, and log level.

Use `examples/config.local.yaml` for local testing and adapt paths for production.

## RBAC

RBAC is configured in the main `config.yaml` and loaded from YAML manifests at server startup:

```yaml
rbac:
  enabled: true
  mode: file
  paths:
    - config/rbac/*.yaml
    - config/rbac/*.yml
```

The database can store applied RBAC metadata internally, but SQL is not the user-facing management workflow. Keep RBAC manifests in Git and deploy them with the rest of the environment configuration.
