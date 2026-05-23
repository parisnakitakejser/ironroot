# Mirroring Helm Charts

Mirror the Helm chart OCI artifact or move the release `.tgz` through your offline package process.

```bash
helm pull oci://ghcr.io/OWNER/charts/ironroot --version 0.1.0
helm push ironroot-0.1.0.tgz oci://registry.internal/charts
```

Use the mirrored chart with mirrored image values.
