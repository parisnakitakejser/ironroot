# Helm Installation

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

IronRoot ships a Helm chart under `deploy/helm/ironroot` and publishes release charts as OCI artifacts.

Install an RC chart:

```bash
helm install ironroot oci://ghcr.io/OWNER/charts/ironroot \
  --version 0.1.0-rc.1 \
  --namespace ironroot \
  --create-namespace \
  --set pki.existingSecret=ironroot-ca \
  --set tls.existingSecret=ironroot-api-tls
```

Install a stable chart:

```bash
helm install ironroot oci://ghcr.io/OWNER/charts/ironroot \
  --version 0.1.0 \
  --namespace ironroot \
  --create-namespace
```

Local chart testing:

```bash
make helm-lint
make helm-template
make helm-test
```

The chart expects CA material from a Kubernetes Secret. Never include the offline Root CA private key.
