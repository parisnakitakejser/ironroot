set shell := ["sh", "-uc"]

go := env_var_or_default("GO", "go")
image := env_var_or_default("IMAGE", "localhost/ironroot:dev")
containerfile := env_var_or_default("CONTAINERFILE", "Containerfile")
helm_chart := env_var_or_default("HELM_CHART", "deploy/helm/ironroot")
install_prefix := env_var_or_default("INSTALL_PREFIX", env_var("HOME") + "/.local")
docs_python := env_var_or_default("DOCS_PYTHON", "/usr/bin/python3")
docs_venv := env_var_or_default("DOCS_VENV", ".venv-docs")
mkdocs := docs_venv + "/bin/mkdocs"

# List available recipes.
default:
    @just --list

# Build current-platform binaries.
build: build-local

# Build current-platform binaries for local development.
build-local:
    {{go}} build -o bin/ironroot-server ./cmd/server
    {{go}} build -o bin/ironroot-admin ./cmd/ironroot-admin
    {{go}} build -o bin/ironroot-client ./cmd/ironroot-client

# Build Linux amd64 and arm64 binaries.
build-linux:
    GOOS=linux GOARCH=amd64 {{go}} build -o dist/linux-amd64/ironroot-server ./cmd/server
    GOOS=linux GOARCH=amd64 {{go}} build -o dist/linux-amd64/ironroot-admin ./cmd/ironroot-admin
    GOOS=linux GOARCH=amd64 {{go}} build -o dist/linux-amd64/ironroot-client ./cmd/ironroot-client
    GOOS=linux GOARCH=arm64 {{go}} build -o dist/linux-arm64/ironroot-server ./cmd/server
    GOOS=linux GOARCH=arm64 {{go}} build -o dist/linux-arm64/ironroot-admin ./cmd/ironroot-admin
    GOOS=linux GOARCH=arm64 {{go}} build -o dist/linux-arm64/ironroot-client ./cmd/ironroot-client

# Build macOS Intel and Apple Silicon binaries.
build-macos:
    GOOS=darwin GOARCH=amd64 {{go}} build -o dist/darwin-amd64/ironroot-server ./cmd/server
    GOOS=darwin GOARCH=amd64 {{go}} build -o dist/darwin-amd64/ironroot-admin ./cmd/ironroot-admin
    GOOS=darwin GOARCH=amd64 {{go}} build -o dist/darwin-amd64/ironroot-client ./cmd/ironroot-client
    GOOS=darwin GOARCH=arm64 {{go}} build -o dist/darwin-arm64/ironroot-server ./cmd/server
    GOOS=darwin GOARCH=arm64 {{go}} build -o dist/darwin-arm64/ironroot-admin ./cmd/ironroot-admin
    GOOS=darwin GOARCH=arm64 {{go}} build -o dist/darwin-arm64/ironroot-client ./cmd/ironroot-client

# Build all supported release targets.
build-all: build-linux build-macos

# Install current-platform binaries into INSTALL_PREFIX/bin.
install-local: build-local
    @echo "Installing freshly built binaries from ./bin into {{install_prefix}}/bin"
    mkdir -p {{install_prefix}}/bin
    cp bin/ironroot-server bin/ironroot-admin bin/ironroot-client {{install_prefix}}/bin/

# Run the local server using .localdev config.
run-server: build-local
    bin/ironroot-server --config .localdev/config/config.yaml

# Smoke-test CLI startup.
smoke-cli: build-local
    bin/ironroot-admin --help >/dev/null
    bin/ironroot-client --help >/dev/null
    bin/ironroot-server --version >/dev/null

# Create local development directories and config.
dev-init:
    mkdir -p .localdev/config .localdev/data .localdev/pki .localdev/certs .localdev/logs
    cp examples/config.local.yaml .localdev/config/config.yaml

# Remove local development state.
dev-clean:
    rm -rf .localdev

# Format Go files.
fmt:
    gofmt -w $(find . -name '*.go' -not -path './vendor/*')

# Fail if Go files need formatting.
fmt-check:
    @unformatted="$(gofmt -l $(find . -name '*.go' -not -path './vendor/*'))"; \
    if [ -n "$unformatted" ]; then \
      echo "gofmt required for:"; \
      echo "$unformatted"; \
      exit 1; \
    fi

# Run go vet.
vet:
    {{go}} vet ./...

# Run golangci-lint.
lint:
    golangci-lint run

# Run unit tests.
test:
    {{go}} test ./...

# Run e2e tests.
test-e2e:
    if [ -d tests/e2e ]; then \
      {{go}} test ./tests/e2e -count=1; \
    else \
      echo "tests/e2e not present; skipping e2e tests"; \
    fi

# Install MkDocs dependencies into the project docs venv.
docs-install:
    if [ ! -x "{{mkdocs}}" ]; then \
      {{docs_python}} -m venv {{docs_venv}}; \
      {{docs_venv}}/bin/python -m pip install -r docs/requirements.txt; \
    fi

docs-setup: docs-install

# Serve docs locally.
docs-serve: docs-setup
    {{mkdocs}} serve

# Build docs with strict validation.
docs-build: docs-setup
    {{mkdocs}} build --strict

# Build generated website output locally.
docs-deploy-local: docs-setup
    {{mkdocs}} build --strict --clean --site-dir site

# Build the container image with Podman or Docker.
container-build:
    if command -v podman >/dev/null 2>&1; then \
      podman build -f {{containerfile}} -t {{image}} .; \
    else \
      docker build -f {{containerfile}} -t {{image}} .; \
    fi

# Run govulncheck.
security-govulncheck:
    govulncheck ./...

# Lint Helm chart.
helm-lint:
    helm lint {{helm_chart}}

# Render Helm chart.
helm-template:
    helm template ironroot {{helm_chart}}

# Package Helm chart.
helm-package:
    mkdir -p dist/charts
    helm package {{helm_chart}} --destination dist/charts

# Run Helm chart tests.
helm-test:
    scripts/helm-test.sh {{helm_chart}}

# Generate Go coverage report.
coverage:
    {{go}} test ./... -coverprofile=coverage.out
    {{go}} tool cover -html=coverage.out -o coverage.html
