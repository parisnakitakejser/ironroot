# Podman Security

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

Prefer rootless Podman. Mount config, data, and PKI material instead of baking secrets into images. Use SELinux volume labels on enforcing hosts and keep PKI mounts read-only where possible.
