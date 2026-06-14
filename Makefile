# Build into bin/ (gitignored) so the binary never collides with the artinstitute/
# source package at the repo root.
BINARY  := bin/artinstitute
PKG     := ./cmd/artinstitute
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X github.com/tamnd/artinstitute-cli/cli.Version=$(VERSION) \
	-X github.com/tamnd/artinstitute-cli/cli.Commit=$(COMMIT) \
	-X github.com/tamnd/artinstitute-cli/cli.Date=$(DATE)

.PHONY: build install test vet fmt clean run

build:
	@mkdir -p $(dir $(BINARY))
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)

install:
	CGO_ENABLED=0 go install -trimpath -ldflags "$(LDFLAGS)" $(PKG)

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w -s .

clean:
	rm -rf bin dist

run: build
	./$(BINARY) $(ARGS)
