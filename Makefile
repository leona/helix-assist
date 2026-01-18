BINARY_NAME=helix-assist

GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

MAIN_PACKAGE=./cmd/helix-assist

BUILD_DIR=build

VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

.PHONY: all build clean test deps help
.PHONY: linux-amd64 linux-arm64 linux-arm darwin-amd64 darwin-arm64 windows-amd64
.PHONY: nixos-amd64 freebsd-amd64 build-all install

all: build

# Build for current platform
build:
	@echo "Building $(BINARY_NAME) for current platform..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Install to $GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GOCMD) install $(LDFLAGS) $(MAIN_PACKAGE)
	@echo "Install complete"

# Build for Linux AMD64
linux-amd64:
	@echo "Building for Linux AMD64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64"

# Build for Linux ARM64
linux-arm64:
	@echo "Building for Linux ARM64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PACKAGE)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64"

# Build for Linux ARM
linux-arm:
	@echo "Building for Linux ARM..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm $(MAIN_PACKAGE)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)-linux-arm"

# Build for macOS AMD64 (Intel)
darwin-amd64:
	@echo "Building for macOS AMD64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64"

# Build for macOS ARM64 (Apple Silicon)
darwin-arm64:
	@echo "Building for macOS ARM64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PACKAGE)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64"

# Build for NixOS AMD64 (statically linked)
nixos-amd64:
	@echo "Building for NixOS AMD64 (static)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-nixos-amd64 $(MAIN_PACKAGE)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)-nixos-amd64"

# Build for FreeBSD AMD64
freebsd-amd64:
	@echo "Building for FreeBSD AMD64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=freebsd GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-freebsd-amd64 $(MAIN_PACKAGE)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)-freebsd-amd64"

# Build for all platforms
build-all: linux-amd64 linux-arm64 linux-arm darwin-amd64 darwin-arm64 nixos-amd64 freebsd-amd64
	@echo "All builds complete"
	@ls -lh $(BUILD_DIR)/

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME)
	@echo "Clean complete"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "Dependencies downloaded"

# Show help
help:
	@echo "Available targets:"
	@echo "  make build          - Build for current platform (default)"
	@echo "  make install        - Install to \$$GOPATH/bin"
	@echo "  make linux-amd64    - Build for Linux AMD64"
	@echo "  make linux-arm64    - Build for Linux ARM64"
	@echo "  make linux-arm      - Build for Linux ARM"
	@echo "  make darwin-amd64   - Build for macOS AMD64 (Intel)"
	@echo "  make darwin-arm64   - Build for macOS ARM64 (Apple Silicon)"
	@echo "  make nixos-amd64    - Build for NixOS AMD64 (static)"
	@echo "  make freebsd-amd64  - Build for FreeBSD AMD64"
	@echo "  make build-all      - Build for all platforms"
	@echo "  make test           - Run tests"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make deps           - Download and tidy dependencies"
	@echo "  make help           - Show this help message"
	@echo ""
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
