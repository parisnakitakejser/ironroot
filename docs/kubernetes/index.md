# Kubernetes

Apply the sample manifests:

```bash
kubectl apply -k deploy/kubernetes
```

The deployment includes:

- Deployment and Service
- Secret for CA material
- ConfigMap for config
- PVC for SQLite data
- Readiness and liveness probes
- Non-root security context
- Dropped Linux capabilities
- OpenTelemetry environment variables

PostgreSQL configuration values are reserved for a future backend implementation.

