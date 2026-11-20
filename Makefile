BINARY := bin/mirolu
PKG := ./cmd/mirolu

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: build build-hardened test test-integration lint fmt secret-scan install-hooks verify clean

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)

build-hardened:
	CGO_ENABLED=0 GOEXPERIMENT=runtimesecret go build -tags runtimesecret -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)

test:
	go test -race -count=1 ./...

test-integration:
	@if [ -d test/integration ]; then \
		go test -race -tags=integration -count=1 ./test/integration/...; \
	else \
		echo "No test/integration suite yet (introduced in Stage 4); skipping."; \
	fi

lint:
	golangci-lint run ./...

fmt:
	golangci-lint fmt ./...

secret-scan:
	gitleaks dir --config .gitleaks.toml --redact .
	@if git rev-parse --verify HEAD >/dev/null 2>&1; then \
		gitleaks git --config .gitleaks.toml --redact .; \
	fi

install-hooks:
	git config core.hooksPath .githooks

verify: lint test build secret-scan

clean:
	rm -rf bin
