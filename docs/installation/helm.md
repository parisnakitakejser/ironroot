# Helm Installation

Install IronRoot into Kubernetes with the Helm chart under `deploy/helm/ironroot`.

```bash
helm install ironroot oci://ghcr.io/OWNER/charts/ironroot \
  --version 0.1.0 \
  --namespace ironroot \
  --create-namespace
```

For Kubernetes-specific chart options, see the Kubernetes section.
