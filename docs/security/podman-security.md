# Podman Security

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

Prefer rootless Podman. Mount config, data, and PKI material instead of baking secrets into images. Use SELinux volume labels on enforcing hosts and keep PKI mounts read-only where possible.
