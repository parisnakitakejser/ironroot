GO ?= go
IMAGE ?= localhost/ironroot:dev

.PHONY: build test test-e2e lint docs-serve docs-build container-build coverage

build:
	$(GO) build -o bin/ironroot-server ./cmd/server
	$(GO) build -o bin/ironroot-admin ./cmd/ironroot-admin
	$(GO) build -o bin/ironroot-client ./cmd/ironroot-client

test:
	$(GO) test ./...

test-e2e:
	$(GO) test ./tests/e2e -count=1

lint:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')
	$(GO) vet ./...

coverage:
	$(GO) test ./... -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html

docs-serve:
	mkdocs serve

docs-build:
	mkdocs build --strict

container-build:
	podman build -f deploy/container/Containerfile -t $(IMAGE) .

