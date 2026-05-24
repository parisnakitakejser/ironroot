# Mirroring the Helm Chart

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

Air-gapped operators should mirror the image and chart through the same approved artifact path used for other infrastructure packages.

Mirror the container image:

```bash
skopeo copy \
  docker://ghcr.io/parisnakitakejser/ironroot:v0.1.0-rc.1 \
  docker://registry.internal/ironroot:v0.1.0-rc.1
```

Mirror the Helm chart:

```bash
helm pull oci://ghcr.io/parisnakitakejser/charts/ironroot --version 0.1.0-rc.1
helm push ironroot-0.1.0-rc.1.tgz oci://registry.internal/charts
```

Install from the internal registry:

```bash
helm install ironroot oci://registry.internal/charts/ironroot \
  --version 0.1.0-rc.1 \
  --namespace ironroot \
  --create-namespace
```

Keep CA material, TLS secrets, and database backups out of image and chart artifacts.
