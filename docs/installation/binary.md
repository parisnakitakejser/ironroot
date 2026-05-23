# Binary Installation

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

Binary deployment is the most direct model for operators who want full control over filesystem permissions, service managers, and backups.

Supported release artifacts:

- `ironroot-linux-amd64.tar.gz`
- `ironroot-linux-arm64.tar.gz`
- `ironroot-darwin-amd64.tar.gz`
- `ironroot-darwin-arm64.tar.gz`

Each archive contains:

- `ironroot-server`
- `ironroot-admin`
- `ironroot-client`

## Linux Install

Download the matching Linux archive, verify `checksums.txt`, and install into `~/.local/bin`:

```bash
mkdir -p ~/.local/bin
tar -xzf ironroot-linux-amd64.tar.gz -C ~/.local/bin
chmod 0755 ~/.local/bin/ironroot-*
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

For arm64 hosts, use `ironroot-linux-arm64.tar.gz`.

## macOS Install

Apple Silicon uses `darwin-arm64`. Intel Macs use `darwin-amd64`.

```bash
mkdir -p ~/.local/bin
tar -xzf ironroot-darwin-arm64.tar.gz -C ~/.local/bin
chmod 0755 ~/.local/bin/ironroot-*
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

If Gatekeeper flags a downloaded binary after checksum verification:

```bash
xattr -d com.apple.quarantine ~/.local/bin/ironroot-admin
xattr -d com.apple.quarantine ~/.local/bin/ironroot-client
xattr -d com.apple.quarantine ~/.local/bin/ironroot-server
```

## Build From Source

Build current-platform binaries:

```bash
make build
```

Build all supported release targets:

```bash
make build-all
```

Output layout:

```text
dist/
  linux-amd64/
  linux-arm64/
  darwin-amd64/
  darwin-arm64/
```

## Recommended Linux Layout

| Path | Contents | Permissions | Backup |
| --- | --- | --- | --- |
| `/etc/ironroot/` | `config.yaml` | directory `0750`, file `0640` or stricter | yes |
| `/var/lib/ironroot/` | SQLite database and application state | `0700`, database `0600` | yes |
| `/var/log/ironroot/` | optional redirected logs | `0750` | policy dependent |
| `/opt/ironroot/` | installed binaries | root-owned, read-only to service user | no, rebuildable |
| `/pki/` | Root certificate, chain, Intermediate cert, encrypted Intermediate key | `0700`, key `0600` | yes |

The Root CA private key does not belong in any online server path.

## macOS Local Layout

For local testing:

| Path | Contents |
| --- | --- |
| `~/.local/bin` | IronRoot binaries |
| `./examples/config.local.yaml` | local config |
| `./data` | local SQLite DB |
| `./pki` | local demo CA material |
| `./certs` | issued demo website certificates |

For production-like macOS hosts, keep config and PKI paths restricted to the service account and use launchd or a process supervisor.

## Trust Stores

macOS Keychain:

```bash
sudo security add-trusted-cert \
  -d \
  -r trustRoot \
  -k /Library/Keychains/System.keychain \
  root-ca.crt
```

Debian/Ubuntu:

```bash
sudo cp root-ca.crt /usr/local/share/ca-certificates/ironroot.crt
sudo update-ca-certificates
```

Fedora/RHEL:

```bash
sudo cp root-ca.crt /etc/pki/ca-trust/source/anchors/ironroot.crt
sudo update-ca-trust
```

Firefox may need manual authority import on both Linux and macOS.

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
