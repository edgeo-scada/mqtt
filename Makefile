# Makefile for edgeo-mqtt CLI

BINARY_NAME := edgeo-mqtt
CMD_PATH := ./cmd/edgeo-mqtt
OUTPUT_DIR := bin

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Go build flags
LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.buildDate=$(BUILD_DATE)

# Default target
.PHONY: all
all: build

# Build for current platform
.PHONY: build
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) $(CMD_PATH)

# Build for all platforms
.PHONY: build-all
build-all: build-linux build-darwin build-windows

# Linux builds
.PHONY: build-linux
build-linux: build-linux-amd64 build-linux-arm64

.PHONY: build-linux-amd64
build-linux-amd64:
	@mkdir -p $(OUTPUT_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(OUTPUT_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_PATH)

.PHONY: build-linux-arm64
build-linux-arm64:
	@mkdir -p $(OUTPUT_DIR)
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(OUTPUT_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_PATH)

# macOS builds
.PHONY: build-darwin
build-darwin: build-darwin-amd64 build-darwin-arm64

.PHONY: build-darwin-amd64
build-darwin-amd64:
	@mkdir -p $(OUTPUT_DIR)
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_PATH)

.PHONY: build-darwin-arm64
build-darwin-arm64:
	@mkdir -p $(OUTPUT_DIR)
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_PATH)

# Windows builds
.PHONY: build-windows
build-windows: build-windows-amd64 build-windows-arm64

.PHONY: build-windows-amd64
build-windows-amd64:
	@mkdir -p $(OUTPUT_DIR)
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(OUTPUT_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_PATH)

.PHONY: build-windows-arm64
build-windows-arm64:
	@mkdir -p $(OUTPUT_DIR)
	GOOS=windows GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(OUTPUT_DIR)/$(BINARY_NAME)-windows-arm64.exe $(CMD_PATH)

# Install locally
.PHONY: install
install:
	go install -ldflags "$(LDFLAGS)" $(CMD_PATH)

# Run tests
.PHONY: test
test:
	go test -v ./...

# Clean build artifacts
.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	rm -rf $(OUTPUT_DIR)

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build              Build for current platform"
	@echo "  build-all          Build for all platforms (Linux, macOS, Windows)"
	@echo "  build-linux        Build for Linux (amd64, arm64)"
	@echo "  build-darwin       Build for macOS (amd64, arm64)"
	@echo "  build-windows      Build for Windows (amd64, arm64)"
	@echo "  install            Install to GOPATH/bin"
	@echo "  test               Run tests"
	@echo "  clean              Remove build artifacts"
	@echo ""
	@echo "Individual platform targets:"
	@echo "  build-linux-amd64    build-linux-arm64"
	@echo "  build-darwin-amd64   build-darwin-arm64"
	@echo "  build-windows-amd64  build-windows-arm64"
