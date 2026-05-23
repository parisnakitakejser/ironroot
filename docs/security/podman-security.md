# Podman Security

Prefer rootless Podman. Mount config, data, and PKI material instead of baking secrets into images. Use SELinux volume labels on enforcing hosts and keep PKI mounts read-only where possible.
