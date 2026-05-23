# Contributing to IronRoot

## Development setup

Install Go, Podman, and Python with MkDocs Material.

```bash
go mod download
make build
```

## Tests

```bash
make test
make test-e2e
make coverage
```

## Documentation

```bash
make docs-serve
make docs-build
```

## Coding conventions

- Keep interfaces small and storage implementations replaceable.
- Do not generate workload private keys on the server.
- Prefer structured logs with component and operation fields.
- Add tests for PKI, enrollment, storage, and telemetry behavior.
- Run `make lint` before opening a pull request.

## Pull requests

Open focused pull requests with clear motivation, tests, and documentation updates when behavior changes.

## Commit messages

Use concise imperative messages, for example:

```text
Add bootstrap token validation
Document root migration flow
```

## Review expectations

Reviews prioritize security boundaries, operational clarity, test coverage, and maintainability.

