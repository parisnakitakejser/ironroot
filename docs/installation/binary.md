# Binary Deployment Architecture

Binary deployment is the most direct model for operators who want full control over filesystem permissions, systemd, and backups.

## Recommended Layout

| Path | Contents | Permissions | Backup |
| --- | --- | --- | --- |
| `/etc/ironroot/` | `config.yaml` | directory `0750`, file `0640` or stricter | yes |
| `/var/lib/ironroot/` | SQLite database and application state | `0700`, database `0600` | yes |
| `/var/log/ironroot/` | optional redirected logs | `0750` | policy dependent |
| `/opt/ironroot/` | installed binaries | root-owned, read-only to service user | no, rebuildable |
| `/pki/` | Root certificate, chain, Intermediate cert, encrypted Intermediate key | `0700`, key `0600` | yes |

The Root CA private key does not belong in any of these paths.

## systemd Example

```ini
[Unit]
Description=IronRoot PKI Server
After=network-online.target

[Service]
User=ironroot
Group=ironroot
Environment=IRONROOT_CONFIG=/etc/ironroot/config.yaml
ExecStart=/opt/ironroot/ironroot-server
Restart=on-failure
NoNewPrivileges=true
ProtectSystem=strict
ReadWritePaths=/var/lib/ironroot
ReadOnlyPaths=/etc/ironroot /pki

[Install]
WantedBy=multi-user.target
```

## Upgrade Workflow

1. Run `ironroot-admin security-check`.
2. Back up `/etc/ironroot`, `/var/lib/ironroot`, and `/pki` excluding any Root private key.
3. Stop the service.
4. Replace binaries in `/opt/ironroot`.
5. Run migrations with `ironroot-admin init-server`.
6. Start the service and verify `/healthz` and `/readyz`.

## Restore Workflow

Restore the database and Intermediate CA material together. A database restored without matching CA material can leave metadata referring to issuers that the server cannot use.

