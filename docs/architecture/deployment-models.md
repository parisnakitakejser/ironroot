# Deployment Models

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

IronRoot can run as a binary, a Podman container, or a Kubernetes workload.

| Model | Responsibility |
| --- | --- |
| Binary | Host owns config, data, logs, and PKI paths. |
| Podman | Image is immutable; config, data, and PKI are mounted. |
| Kubernetes | Deployment, Secret, ConfigMap, PVC, Service, and optional ServiceMonitor manage runtime state. |

All models keep the Root CA private key offline.
