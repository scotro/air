BINARY := air
BUILD_DIR := bin
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -X main.version=dev -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: build clean test

build:
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) ./cmd/air/

test:
	go test ./...

clean:
	rm -rf $(BUILD_DIR)
