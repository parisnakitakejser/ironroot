# CLI

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

IronRoot ships three command surfaces:

- `ironroot-server`
- `ironroot-admin`
- `ironroot-client`

Optional future aliases are `ironrootd`, `ironctl`, and `iron-agent`.

## Admin examples

```bash
ironroot-admin init-server
ironroot-admin create-token --host node-01 --ttl 1h
ironroot-admin revoke-cert --server http://localhost:8443 --serial <serial>
ironroot-admin migration-status
```

## Client examples

```bash
ironroot-client trust install --server http://localhost:8443 --out trust
ironroot-client enroll --server http://localhost:8443 --token <token>
ironroot-client request-cert --server http://localhost:8443 --enrollment-id <id> --dns app.internal --out certs
ironroot-client renew --server http://localhost:8443 --enrollment-id <id> --dns app.internal --out certs
ironroot-client status --server http://localhost:8443 --serial <serial>
```

