.PHONY: all build test lint fmt vet check clean

BINARY_NAME=gort
BUILD_DIR=bin

all: check build

build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/gort

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	CGO_ENABLED=0 go test -v ./...

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, skipping (go vet passed)"; \
	fi

check: fmt vet test lint

clean:
	rm -rf $(BUILD_DIR)
