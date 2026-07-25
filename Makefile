GO ?= go
NPM ?= npm
VERSION := $(shell tr -d '\r\n' < VERSION)
LDFLAGS := -s -w -X github.com/kejilion/kejilion-panel/internal/version.Version=$(VERSION)

.PHONY: fmt test test-go test-web build build-web build-linux clean

fmt:
	$(GO) fmt ./...

test: test-go test-web

test-go:
	$(GO) test ./...

test-web:
	cd web && $(NPM) run typecheck && $(NPM) test

build: build-web
	mkdir -p dist
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o dist/paneld ./cmd/paneld
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o dist/kejilion-agent ./cmd/kejilion-agent
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o dist/kpctl ./cmd/kpctl

build-web:
	cd web && $(NPM) run build

build-linux: build-web
	mkdir -p dist/linux-amd64 dist/linux-arm64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o dist/linux-amd64/paneld ./cmd/paneld
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o dist/linux-amd64/kejilion-agent ./cmd/kejilion-agent
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o dist/linux-amd64/kpctl ./cmd/kpctl
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o dist/linux-arm64/paneld ./cmd/paneld
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o dist/linux-arm64/kejilion-agent ./cmd/kejilion-agent
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o dist/linux-arm64/kpctl ./cmd/kpctl

clean:
	rm -rf dist web/dist coverage.out

