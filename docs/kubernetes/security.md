# Kubernetes Security

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

The Helm chart defaults to non-root execution, dropped capabilities, ClusterIP Service, and mounted CA material. Enable NetworkPolicy and restrict Secret access in production.

The Root CA private key should never be copied into the cluster.
