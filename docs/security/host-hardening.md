# Host Hardening

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

IronRoot should run on a small, controlled host or Kubernetes workload with clear ownership of config, data, and PKI material.

Recommended host posture:

- run as a dedicated non-root user
- restrict config, data, and PKI directories to the IronRoot user
- use `0600` for private keys and SQLite database files
- mount CA material from `/pki` instead of baking it into images
- mount config from `/config/config.yaml`
- mount data from `/data`
- enable API TLS before binding to public interfaces
- back up the database and Intermediate CA material together
- keep audit logs writable and retained

For Kubernetes:

- use a dedicated namespace
- run as non-root
- enable `readOnlyRootFilesystem` where possible
- drop all Linux capabilities
- store CA material in a Secret
- use a PVC for SQLite data
- configure liveness and readiness probes
- add a NetworkPolicy for the API server

