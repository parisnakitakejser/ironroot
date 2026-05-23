GO ?= go
IMAGE ?= localhost/ironroot:dev
CONTAINERFILE ?= Containerfile
GOFILES := $(shell find . -name '*.go' -not -path './vendor/*')

.PHONY: build fmt fmt-check vet lint test test-e2e docs-serve docs-build container-build security-govulncheck coverage

build:
	$(GO) build -o bin/ironroot-server ./cmd/server
	$(GO) build -o bin/ironroot-admin ./cmd/ironroot-admin
	$(GO) build -o bin/ironroot-client ./cmd/ironroot-client

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

docs-serve:
	mkdocs serve

docs-build:
	mkdocs build --strict

container-build:
	@if command -v podman >/dev/null 2>&1; then \
		podman build -f $(CONTAINERFILE) -t $(IMAGE) .; \
	else \
		docker build -f $(CONTAINERFILE) -t $(IMAGE) .; \
	fi

security-govulncheck:
	govulncheck ./...

coverage:
	$(GO) test ./... -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
