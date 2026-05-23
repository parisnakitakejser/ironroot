# Local Development

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

IronRoot supports development on Linux and macOS.

Supported local and release targets:

- Linux amd64
- Linux arm64
- macOS Intel amd64
- macOS Apple Silicon arm64

Windows support is planned, but not supported in the MVP.

## Linux Setup

Install Go, Git, SQLite development libraries if your distribution requires them for CGO, and either Podman or Docker.

```bash
make build-local
make test
make build-linux
```

Install local binaries:

```bash
make install-local
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

`make install-local` and `just install-local` rebuild current-platform binaries before installing them, so the installed commands match the latest checkout.

For zsh:

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

## macOS Setup

Install Go and Git. Homebrew is optional but useful for `podman`, `docker`, `sqlite`, and documentation tooling.

```bash
make build-local
make test
make build-macos
```

Apple Silicon uses the `darwin-arm64` target. Intel Macs use `darwin-amd64`.

Install local binaries:

```bash
make install-local
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

`/usr/local/bin` is also common on Intel Macs and `/opt/homebrew/bin` is common on Apple Silicon, but `~/.local/bin` works without elevated privileges.

If macOS Gatekeeper flags a downloaded release binary, remove quarantine after verifying the checksum:

```bash
xattr -d com.apple.quarantine ~/.local/bin/ironroot-admin
```

## Local Binary Install

`make install-local` installs:

- `ironroot-server`
- `ironroot-admin`
- `ironroot-client`

into:

```text
~/.local/bin
```

Override with:

```bash
make install-local INSTALL_PREFIX=/usr/local
```

## PATH Setup

Linux bash:

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

macOS zsh:

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

## Local TLS Trust

macOS:

```bash
sudo security add-trusted-cert \
  -d \
  -r trustRoot \
  -k /Library/Keychains/System.keychain \
  ./pki/root/root-ca.crt
```

Debian/Ubuntu:

```bash
sudo cp ./pki/root/root-ca.crt /usr/local/share/ca-certificates/ironroot-local.crt
sudo update-ca-certificates
```

Fedora/RHEL:

```bash
sudo cp ./pki/root/root-ca.crt /etc/pki/ca-trust/source/anchors/ironroot-local.crt
sudo update-ca-trust
```

Firefox may use its own trust store on both Linux and macOS. Import the Root CA under certificate authorities if browser trust does not follow the OS trust store.

## Debugging Locally

Run the server with a local config:

```bash
make run-server
```

Use separate terminals for admin and client commands:

```bash
ironroot-admin --config ./examples/config.local.yaml create-token --host local-demo --ttl 24h
ironroot-client enroll --server http://localhost:8443 --hostname local-demo --token <token>
```

On Linux, inspect file permissions with `stat` or `ls -l`. On macOS, use `stat -f "%Sp %N"` for BSD-style output.

## Podman and Docker Notes

Linux:

- Podman works well rootless.
- Docker is also supported for local builds.

macOS:

- Use Podman Desktop or Docker Desktop.
- Container networking differs from Linux; use `host.containers.internal` for host services from containers where supported.

## Local Telemetry Testing

Start the local observability stack from `examples/otel/`, then run IronRoot with:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
OTEL_SERVICE_NAME=ironroot-local \
ironroot-server --config ./examples/config.local.yaml
```
