VERSION := 0.1.0
BINARY := air
BUILD_DIR := bin

.PHONY: build clean

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/air/

clean:
	rm -rf $(BUILD_DIR)
