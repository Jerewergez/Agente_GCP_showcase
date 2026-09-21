# Agente_GCP — Go Backend Makefile
# Targets: build, test, lint, run, clean

GO ?= go
GOFLAGS ?=
BINARY ?= agente-gcp
MAIN_PKG ?= .

# Detect OS for binary name
ifeq ($(OS),Windows_NT)
	BINARY := $(BINARY).exe
endif

.PHONY: all build test lint run clean help

all: build

build:
	$(GO) build $(GOFLAGS) -o $(BINARY) $(MAIN_PKG)

test:
	$(GO) test $(GOFLAGS) -v -race -count=1 ./...

lint:
	@golangci-lint run ./... 2>/dev/null || (echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; exit 1)

run: build
	./$(BINARY)

clean:
	-rm -f $(BINARY)
	-go clean -cache

help:
	@echo "Targets:"
	@echo "  build  - Build the server binary"
	@echo "  test   - Run all tests with race detection"
	@echo "  lint   - Run golangci-lint"
	@echo "  run    - Build and run the server"
	@echo "  clean  - Remove build artifacts and cache"
