GO ?= go
IMAGE ?= localhost/ironroot:dev
CONTAINERFILE ?= Containerfile
HELM_CHART ?= deploy/helm/ironroot
INSTALL_PREFIX ?= $(HOME)/.local
DOCS_PYTHON ?= /usr/bin/python3
DOCS_VENV ?= .venv-docs
MKDOCS ?= $(DOCS_VENV)/bin/mkdocs
GOFILES := $(shell find . -name '*.go' -not -path './vendor/*')
BINARIES := ironroot-server ironroot-admin ironroot-client
PLATFORMS := linux-amd64 linux-arm64 darwin-amd64 darwin-arm64

.PHONY: build build-local build-linux build-macos build-all install-local run-server smoke-cli dev-init dev-clean fmt fmt-check vet lint test test-e2e docs-install docs-setup docs-serve docs-build docs-deploy-local container-build security-govulncheck helm-lint helm-template helm-package helm-test coverage

build: build-local

build-local:
	$(GO) build -o bin/ironroot-server ./cmd/server
	$(GO) build -o bin/ironroot-admin ./cmd/ironroot-admin
	$(GO) build -o bin/ironroot-client ./cmd/ironroot-client

build-linux:
	GOOS=linux GOARCH=amd64 $(GO) build -o dist/linux-amd64/ironroot-server ./cmd/server
	GOOS=linux GOARCH=amd64 $(GO) build -o dist/linux-amd64/ironroot-admin ./cmd/ironroot-admin
	GOOS=linux GOARCH=amd64 $(GO) build -o dist/linux-amd64/ironroot-client ./cmd/ironroot-client
	GOOS=linux GOARCH=arm64 $(GO) build -o dist/linux-arm64/ironroot-server ./cmd/server
	GOOS=linux GOARCH=arm64 $(GO) build -o dist/linux-arm64/ironroot-admin ./cmd/ironroot-admin
	GOOS=linux GOARCH=arm64 $(GO) build -o dist/linux-arm64/ironroot-client ./cmd/ironroot-client

build-macos:
	GOOS=darwin GOARCH=amd64 $(GO) build -o dist/darwin-amd64/ironroot-server ./cmd/server
	GOOS=darwin GOARCH=amd64 $(GO) build -o dist/darwin-amd64/ironroot-admin ./cmd/ironroot-admin
	GOOS=darwin GOARCH=amd64 $(GO) build -o dist/darwin-amd64/ironroot-client ./cmd/ironroot-client
	GOOS=darwin GOARCH=arm64 $(GO) build -o dist/darwin-arm64/ironroot-server ./cmd/server
	GOOS=darwin GOARCH=arm64 $(GO) build -o dist/darwin-arm64/ironroot-admin ./cmd/ironroot-admin
	GOOS=darwin GOARCH=arm64 $(GO) build -o dist/darwin-arm64/ironroot-client ./cmd/ironroot-client

build-all: build-linux build-macos

install-local: build-local
	mkdir -p $(INSTALL_PREFIX)/bin
	cp bin/ironroot-server bin/ironroot-admin bin/ironroot-client $(INSTALL_PREFIX)/bin/

run-server: build-local
	bin/ironroot-server --config .localdev/config/config.yaml

smoke-cli: build-local
	bin/ironroot-admin --help >/dev/null
	bin/ironroot-client --help >/dev/null
	bin/ironroot-server --version >/dev/null

dev-init:
	mkdir -p .localdev/config .localdev/data .localdev/pki .localdev/certs .localdev/logs
	cp examples/config.local.yaml .localdev/config/config.yaml

dev-clean:
	rm -rf .localdev

fmt:
	gofmt -w $(GOFILES)

fmt-check:
	@UNFORMATTED="$$(gofmt -l $(GOFILES))"; \
	if [ -n "$$UNFORMATTED" ]; then \
		echo "gofmt required for:"; \
		echo "$$UNFORMATTED"; \
		exit 1; \
	fi

vet:
	$(GO) vet ./...

lint:
	golangci-lint run

test:
	$(GO) test ./...

test-e2e:
	@if [ -d tests/e2e ]; then \
		$(GO) test ./tests/e2e -count=1; \
	else \
		echo "tests/e2e not present; skipping e2e tests"; \
	fi

docs-install:
	@if [ ! -x "$(MKDOCS)" ]; then \
		$(DOCS_PYTHON) -m venv $(DOCS_VENV); \
		$(DOCS_VENV)/bin/python -m pip install -r docs/requirements.txt; \
	fi

docs-setup: docs-install

docs-serve: docs-setup
	$(MKDOCS) serve

docs-build: docs-setup
	$(MKDOCS) build --strict

docs-deploy-local: docs-setup
	$(MKDOCS) build --strict --clean --site-dir site

container-build:
	@if command -v podman >/dev/null 2>&1; then \
		podman build -f $(CONTAINERFILE) -t $(IMAGE) .; \
	else \
		docker build -f $(CONTAINERFILE) -t $(IMAGE) .; \
	fi

security-govulncheck:
	govulncheck ./...

helm-lint:
	helm lint $(HELM_CHART)

helm-template:
	helm template ironroot $(HELM_CHART)

helm-package:
	mkdir -p dist/charts
	helm package $(HELM_CHART) --destination dist/charts

helm-test:
	scripts/helm-test.sh $(HELM_CHART)

coverage:
	$(GO) test ./... -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
