# ironroot-client

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

`ironroot-client` enrolls machines, generates private keys locally, sends CSRs to IronRoot, and writes certificates in a service-friendly layout.

## Enroll

```bash
ironroot-client enroll \
  --server http://localhost:8443 \
  --hostname demo.local \
  --token <bootstrap-token>
```

`--hostname` must match the host restriction on the bootstrap token. The response contains an `enrollment_id`; use that UUID for certificate requests.

## Request Certificate

```bash
ironroot-client request-cert \
  --server http://localhost:8443 \
  --enrollment-id <enrollment_id> \
  --dns demo.local \
  --out .localdev/certs/demo.local
```

If `--out` is omitted, IronRoot uses the platform certificate directory:

| Platform | Default path |
|---|---|
| Linux | `~/.local/share/ironroot/certs/<dns-name>/` |
| macOS | `~/Library/Application Support/ironroot/certs/<dns-name>/` |

Generated files:

| File | Sensitive | Purpose |
|---|---:|---|
| `tls.key` | Yes | Locally generated private key. Never share it. |
| `tls.crt` | No | Issued server certificate. |
| `ca-chain.crt` | No | Intermediate and Root CA chain. |
| `fullchain.crt` | No | `tls.crt` plus CA chain for nginx and Caddy. |
| `metadata.json` | No | DNS names, enrollment ID, serial, expiry, key type, and fingerprint. |
| `fingerprints.txt` | No | SHA-256 fingerprints for debugging. |
| `README.txt` | No | Local usage notes. |

## request-cert Flags

| Flag | Default | Description |
|---|---|---|
| `--server` | `http://localhost:8443` | IronRoot API URL. |
| `--enrollment-id` | required | UUID returned by `ironroot-client enroll`. |
| `--dns` | required | Comma-separated DNS names for the certificate. |
| `--out` | platform-specific | Output directory. Created automatically. |
| `--overwrite` | `false` | Replace existing generated files. |
| `--bundle` | `true` | Generate `fullchain.crt`. |
| `--key-type` | `ecdsa` | Private key type: `ecdsa` or `rsa`. |
| `--curve` | `p256` | ECDSA curve: `p256`, `p384`, or `p521`. |
| `--rsa-bits` | `2048` | RSA size when `--key-type rsa`. |
| `--format` | `pem` | Output format. Only PEM is currently supported. |
| `--key-password` | empty | Encrypt `tls.key` with this password. Avoid shell history in production. |
| `--key-password-env` | empty | Environment variable containing key password. |
| `--chmod` | mixed | Override file mode. Defaults are `0600` for `tls.key`, `0644` for cert files. |
| `--owner` | empty | Optional owner name or uid for generated files. |
| `--group` | empty | Optional group name or gid for generated files. |

## Security Model

The client generates the private key locally. IronRoot receives only the CSR. The private key is never uploaded to the server.

By default, `tls.key` is written with `0600` permissions and existing files are not overwritten unless `--overwrite` is provided.
