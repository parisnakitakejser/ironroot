# Kubernetes Security

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

Use non-root pods, read-only root filesystems where possible, dropped capabilities, restricted Secret access, PVCs for SQLite, NetworkPolicy, and internal Services by default.

Never store the Root CA private key in Kubernetes.
