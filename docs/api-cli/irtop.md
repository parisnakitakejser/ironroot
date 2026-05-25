# irtop

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

`irtop` is **IronRoot Top**, a read-only terminal UI for operators who need a fast view of IronRoot server health, CA health, certificate lifecycle activity, enrollments, bootstrap tokens, audit events, security posture, and telemetry.

It is separate from the production server and automation CLIs:

- `ironroot-server` runs the PKI API.
- `ironroot-admin` performs administrative workflows.
- `ironroot-client` performs enrollment and certificate requests.
- `ironroot-dev` supports contributor workflows.
- `irtop` monitors the platform in real time.

## Install

For local development, build and install all current-platform binaries:

```bash
just install-local
```

Verify:

```bash
irtop --help
```

## Connect

Production default:

```bash
irtop --server https://ironroot.example.com:8443 --ca-file ./root-ca.crt
```

Local development with the default `.localdev` config:

```bash
irtop --server http://localhost:8443
```

!!! warning
    The URL scheme must match the IronRoot server. `irtop` will not silently downgrade HTTPS to HTTP.

If the server uses HTTPS with a private Root CA, pass the Root CA or CA bundle:

```bash
irtop --server https://localhost:8443 \
  --ca-file .localdev/pki/root/root-ca.crt
```

Use `--insecure-skip-verify` only for deliberate TLS troubleshooting:

```bash
irtop --server https://localhost:8443 --insecure-skip-verify
```

## Text Mode

Use text output when you want a one-shot status check in scripts or CI:

```bash
irtop --server http://localhost:8443 --output text
```

## Configuration

`irtop` loads YAML configuration from `~/.ironroot/config` by default. The config directory is `~/.ironroot` and the default filename is `config`.

```bash
mkdir -p ~/.ironroot
cp examples/irtop.config ~/.ironroot/config
irtop
```

For quick local checks, `--server` can be used without creating `~/.ironroot/config` first:

```bash
irtop --server http://localhost:8443
```

Config example:

```yaml
default_profile: local
profiles:
  local:
    endpoint: http://localhost:8443
    refresh: 5s
    default_view: overview
    output: tui
  production:
    endpoint: https://ironroot.example.com:8443
    ca_file: ~/ironroot/root-ca.crt
    refresh: 10s
    default_view: security
    output: tui
```

When more than one profile is configured, `irtop` shows the active profile in the header and lets you switch profiles inside the TUI without restarting. Profile switches rebuild the API client and reload data for the newly selected profile.

Use `--profile` to choose the startup profile instead of `default_profile`:

```bash
irtop --profile production
irtop --config ./examples/irtop.config --profile local
```

Use `--config` only when you intentionally want a different YAML file:

```bash
irtop --config ./examples/irtop.config
```

The config file must use the `profiles` map. Each profile must include `server` or `endpoint`, plus `refresh`, `default_view`, and `output`. Unknown YAML fields, duplicate profile names, invalid profile shapes, invalid YAML, missing files, and missing required values are reported as startup errors.

## TUI Controls

| Key | Action |
|---|---|
| `p` | Open or close the profile selector. |
| `up` / `down` | Move through profiles in the selector. |
| `k` / `j` | Move through profiles in the selector. |
| `enter` | Switch to the selected profile and reload data. |
| `[` / `]` | Switch directly to the previous or next profile. |
| `r` | Refresh the active profile. |

## Flags

| Flag | Default | Description |
|---|---:|---|
| `--server` | config value | IronRoot API server URL. |
| `--config` | `~/.ironroot/config` | Optional override for the default `irtop` YAML config path. |
| `--profile` | `default_profile` | Profile to select at startup. |
| `--ca-file` | empty | CA bundle used to verify HTTPS connections. |
| `--refresh` | `5s` | Dashboard refresh interval. |
| `--insecure-skip-verify` | `false` | Skip TLS certificate verification. |
| `--token` | empty | Read-only admin token. Token values are never displayed in the UI. |
| `--output` | `tui` | Output mode: `tui` or `text`. |

## HTTP vs HTTPS

`irtop` accepts both `http://` and `https://` URLs, but it requires the scheme to be explicit.

- Use `http://localhost:8443` only for local development when the server TLS config is empty.
- Use `https://...` for production, Kubernetes, Podman, and shared environments.
- Use `--ca-file` when the server certificate chains to a private IronRoot Root CA that is not in the system trust store.
- Do not use `--insecure-skip-verify` as a normal workflow.

If you connect with `https://localhost:8443` while the server is speaking HTTP, `irtop` prints an actionable message and suggests:

```bash
irtop --server http://localhost:8443
```

## Kubernetes Port-Forward

```bash
kubectl -n ironroot port-forward svc/ironroot 8443:8443
irtop --server https://localhost:8443 \
  --ca-file ./root-ca.crt
```

## Views

`irtop` includes these read-only views:

- Overview
- Certificates
- Enrollments
- Tokens
- CA Health
- Security
- Telemetry
- Audit Log
- Server
- Help

## Keyboard Shortcuts

| Key | Action |
|---|---|
| `q` | Quit |
| `?` | Help |
| `r` | Refresh now |
| `1` | Overview |
| `2` | Certificates |
| `3` | Enrollments |
| `4` | Tokens |
| `5` | CA Health |
| `6` | Security |
| `7` | Telemetry |
| `8` | Audit Log |
| `9` | Server |
| `/` | Search/filter placeholder |
| `s` | Sort placeholder |
| `enter` | Details placeholder |
| `esc` | Back |

## Security Model

`irtop` is read-only in the first implementation. It does not revoke certificates, create tokens, delete resources, or mutate server state.

The status API used by `irtop` is designed to avoid sensitive material:

- no private keys
- no bootstrap token secret values
- no token hashes
- no raw CA key material
- no destructive operations

Use a read-only admin credential when authentication support is enabled.
