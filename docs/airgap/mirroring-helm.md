# Mirroring Helm Charts

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

Mirror the Helm chart OCI artifact or move the release `.tgz` through your offline package process.

```bash
helm pull oci://ghcr.io/parisnakitakejser/charts/ironroot --version 0.1.0
helm push ironroot-0.1.0.tgz oci://registry.internal/charts
```

Use the mirrored chart with mirrored image values.
