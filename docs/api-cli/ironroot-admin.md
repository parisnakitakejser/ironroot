# ironroot-admin

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

`ironroot-admin` is the trusted operator CLI. Use it from secure operator workstations for bootstrap, CA management, token creation, security checks, and migration workflows.

## Root CA Creation

Beginner mode:

```bash
ironroot-admin ca create-root \
  --name "IronRoot Local Root CA" \
  --out .localdev/pki/root
```

Advanced mode:

```bash
ironroot-admin ca create-root \
  --name "IronRoot Production Root CA" \
  --common-name "IronRoot Root CA" \
  --organization "IronRoot" \
  --organizational-unit "Security" \
  --country "DK" \
  --province "Hovedstaden" \
  --locality "Copenhagen" \
  --algorithm ecdsa \
  --curve p384 \
  --rsa-bits 4096 \
  --validity 20y \
  --max-path-length 1 \
  --key-password-env ROOT_CA_PASSWORD \
  --encrypt-key \
  --crl-enabled \
  --ocsp-enabled \
  --allow-server-auth=false \
  --allow-client-auth=false \
  --allow-code-signing=false \
  --allow-email-protection=false \
  --offline \
  --out ./pki/root
```

## Root CA Flags

| Argument | Default | Description | Recommendation |
|---|---|---|---|
| `--name` | `IronRoot Local Root CA` | Friendly Root CA name. | Use a clear environment-specific name. |
| `--common-name` | value of `--name` | Certificate subject CN. | Keep stable for the trust domain. |
| `--organization` | `IronRoot` | Subject organization. | Set to your org for production. |
| `--organizational-unit` | empty | Subject OU. | Optional security/team label. |
| `--country` | empty | Subject country code. | Optional. |
| `--province` | empty | Subject state/province. | Optional. |
| `--locality` | empty | Subject locality/city. | Optional. |
| `--algorithm` | `ecdsa` | Root private key algorithm. | Use `ecdsa` unless policy requires RSA. |
| `--rsa-bits` | `4096` | RSA size when `--algorithm rsa`. | Use `4096` for Root CAs. |
| `--curve` | `p384` | ECDSA curve. | Use `p384` for Root CAs. |
| `--key-format` | `pkcs8` | Private key encoding. | Keep `pkcs8`. |
| `--encrypt-key` | `true` | Encrypt private key at rest. | Keep enabled. |
| `--key-password` | empty | Password value. | Avoid in shell history for production. |
| `--key-password-env` | empty | Password environment variable. | Prefer for automation. |
| `--key-password-file` | empty | Password file. | Use only with strict permissions. |
| `--validity` | `20y` | Root CA lifetime. | 20 years with planned migration. |
| `--not-before` | now minus one minute | RFC3339 start time. | Usually omit. |
| `--max-path-length` | `1` | CA depth below Root. | Keep `1` for Root → Intermediate → leaf. |
| `--is-ca` | `true` | Marks certificate as a CA. | Must stay true. |
| `--allow-cert-signing` | `true` | Allows signing Intermediate CAs. | Keep enabled. |
| `--allow-crl-signing` | `true` | Allows CRL signing. | Keep enabled for revocation readiness. |
| `--allow-server-auth` | `false` | Adds serverAuth EKU. | Keep disabled for Root CAs. |
| `--allow-client-auth` | `false` | Adds clientAuth EKU. | Keep disabled for Root CAs. |
| `--allow-code-signing` | `false` | Adds codeSigning EKU. | Keep disabled unless designing a separate CA. |
| `--allow-email-protection` | `false` | Adds emailProtection EKU. | Keep disabled unless designing a separate CA. |
| `--generate-trust-bundle` | `true` | Writes public trust bundle. | Keep enabled. |
| `--pem` | `true` | Writes PEM certificate export. | Keep enabled. |
| `--der` | `true` | Writes DER certificate export. | Useful for some trust stores. |
| `--offline` | `true` | Records offline custody intent. | Keep enabled for production. |
| `--backup-reminder` | `true` | Writes recovery reminders. | Keep enabled. |
| `--crl-enabled` | `false` | Future CRL metadata. | Enable when planning CRL operations. |
| `--ocsp-enabled` | `false` | Future OCSP metadata. | Enable when planning OCSP operations. |
| `--permit-dns` | empty | DNS name constraints. | Advanced only. |
| `--permit-ip` | empty | IP CIDR name constraints. | Advanced only. |
| `--permit-email` | empty | Email name constraints. | Advanced only. |
| `--out` | `./pki/root` | Output directory. | Use an offline storage path in production. |

## Inspect CA Material

```bash
ironroot-admin ca inspect --output table ./pki/root/root-ca.crt
ironroot-admin ca inspect --output markdown ./pki/root/root-ca.crt
ironroot-admin ca inspect --output json ./pki/root/root-ca.crt
```

## Common Commands

```bash
ironroot-admin ca create-intermediate --root-cert ./pki/root/root-ca.crt --root-key ./pki/root/root-ca.key --out ./pki/intermediate
ironroot-admin bootstrap --config ./examples/config.local.yaml
ironroot-admin --config ./examples/config.local.yaml create-token --host local-demo --ttl 24h
ironroot-admin --config ./examples/config.local.yaml list-tokens
ironroot-admin --config ./examples/config.local.yaml inspect-token <token-id>
ironroot-admin --config ./examples/config.local.yaml revoke-token <token-id>
ironroot-admin security-check --config ./examples/config.local.yaml
```

## Bootstrap Token Operations

`create-token` prints the token secret once and stores only a hash in the database. `list-tokens` and `inspect-token` never print token secrets.

```bash
ironroot-admin --config ./examples/config.local.yaml list-tokens --active
ironroot-admin --config ./examples/config.local.yaml list-tokens --host demo.local --wide
ironroot-admin --config ./examples/config.local.yaml list-tokens --json
ironroot-admin --config ./examples/config.local.yaml list-tokens --markdown
```

Token statuses are:

| Status | Meaning |
|---|---|
| `active` | Token is unexpired, not revoked, and unused. |
| `used` | Token has created at least one enrollment. |
| `expired` | Token is past `expires_at`. |
| `revoked` | Token was explicitly revoked. |
