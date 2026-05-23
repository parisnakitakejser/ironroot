# Helm Installation

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

Install IronRoot into Kubernetes with the Helm chart under `deploy/helm/ironroot`.

```bash
helm install ironroot oci://ghcr.io/OWNER/charts/ironroot \
  --version 0.1.0 \
  --namespace ironroot \
  --create-namespace
```

For Kubernetes-specific chart options, see the Kubernetes section.
