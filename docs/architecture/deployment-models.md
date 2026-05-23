# Deployment Models

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

IronRoot can run as a binary, a Podman container, or a Kubernetes workload.

| Model | Responsibility |
| --- | --- |
| Binary | Host owns config, data, logs, and PKI paths. |
| Podman | Image is immutable; config, data, and PKI are mounted. |
| Kubernetes | Deployment, Secret, ConfigMap, PVC, Service, and optional ServiceMonitor manage runtime state. |

All models keep the Root CA private key offline.
