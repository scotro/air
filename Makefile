VERSION := 0.1.0
BINARY := air
BUILD_DIR := bin

.PHONY: build install clean

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/air/

install: build
	cp $(BUILD_DIR)/$(BINARY) /usr/local/bin/

clean:
	rm -rf $(BUILD_DIR)
