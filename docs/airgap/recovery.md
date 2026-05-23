# Airgap Recovery

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

Recovery depends on what was lost:

- Lost server database: restore the SQLite or PostgreSQL backup.
- Lost Intermediate CA key: restore encrypted Intermediate CA backup or create a new Intermediate from the offline Root.
- Lost Root CA key: recover from offline backups. If unavailable, create a new Root and migrate trust.

Never improvise by copying Root CA private keys into online systems during an incident.
