# ironroot-dev

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

`ironroot-dev` is the contributor-only helper CLI for local IronRoot development workflows. It is installed by `just install-local` and is not required for production deployments.

The CLI exists to keep local developer workflows versioned, testable, and easier to extend than shell-only task runner logic.

!!! warning "Developer source build required"

    `ironroot-dev` is only available after building IronRoot from the Git repository. It is not a production runtime dependency and should not be expected on servers, containers, or Kubernetes deployments. Run `just install-local` from a source checkout to build and install it into your local PATH.

## Install

Build and install all local binaries:

```bash
just install-local
```

Verify the developer CLI:

```bash
ironroot-dev --help
ironroot-dev dev-init --help
```

## Commands

| Command | Purpose |
|---|---|
| `ironroot-dev dev-init` | Initialize a local `.localdev` workspace. |
| `ironroot-dev completion` | Generate shell completion scripts. |
| `ironroot-dev help` | Show command help. |

Future developer helpers may include local environment checks, demo helpers, docs helpers, and debugging workflows.

## dev-init

`dev-init` creates a neutral local development workspace. It is self-contained: it does not need to read `examples/config.local.yaml`, and it does not need to run from inside an IronRoot Git checkout.

Default usage:

```bash
ironroot-dev dev-init
```

Create the workspace under another base directory:

```bash
ironroot-dev dev-init --base-dir /path/to/workspace
```

Preview actions without writing files:

```bash
ironroot-dev dev-init --dry-run --verbose
```

Regenerate generated helper files:

```bash
ironroot-dev dev-init --force
```

## dev-init Flags

| Flag | Default | Description |
|---|---|---|
| `--base-dir` | current directory | Base directory for the local workspace. This does not need to be a Git checkout. |
| `--output` | `.localdev` | Workspace path. Relative paths are resolved inside the base directory. |
| `--force` | `false` | Overwrite generated `config.yaml`, `.gitignore`, and `README.txt` if they already exist. |
| `--dry-run` | `false` | Print actions without creating directories or writing files. |
| `--verbose` | `false` | Print detailed path and file actions, including existing directories. |
| `--help` | `false` | Show help for the command. |

## Generated Layout

`dev-init` creates:

```text
.localdev/
  config/
    config.yaml
  data/
  pki/
    root/
    intermediate/
  certs/
  logs/
  tmp/
  .gitignore
  README.txt
```

Generated paths in `.localdev/config/config.yaml` are absolute, so the config can be used even when `ironroot-server` or `ironroot-admin` is launched from another directory.

## Overwrite Behavior

`dev-init` is safe to run more than once.

- Existing directories are reused.
- Existing generated files are skipped by default.
- Use `--force` only when you intentionally want to regenerate the local config and helper files.

The command never creates a demo DNS name or host-specific certificate directory. Certificate output directories are created later by `ironroot-client request-cert`.

## Production Boundary

`ironroot-dev` is not a production binary. Production deployments should use:

- `ironroot-server`
- `ironroot-admin`
- `ironroot-client`

Do not depend on `ironroot-dev` in Kubernetes, Podman, or server runtime automation.
