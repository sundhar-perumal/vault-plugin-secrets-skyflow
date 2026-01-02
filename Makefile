# Makefile for Skyflow Vault Plugin

# Variables
BINARY_NAME=skyflow-plugin
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DIR=bin
PLUGIN_DIR=/etc/vault/plugins
GO_FILES=$(shell find . -name '*.go' -type f -not -path "./vendor/*")
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -s -w"
BUILD_FLAGS=-trimpath

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
BLUE=\033[0;34m
NC=\033[0m # No Color

.PHONY: all build clean test coverage fmt vet lint install uninstall help
.PHONY: build-all build-linux build-darwin build-windows
.PHONY: test-unit test-integration test-all test-verbose test-race
.PHONY: docker-build docker-run
.PHONY: release-check version

.DEFAULT_GOAL := help

## help: Display this help message
help:
	@echo "$(BLUE)Skyflow Vault Plugin - Makefile Commands$(NC)"
	@echo ""
	@echo "$(GREEN)Build Commands:$(NC)"
	@echo "  make build              - Build the plugin for current OS/arch"
	@echo "  make build-all          - Build for all platforms (Linux, macOS, Windows)"
	@echo "  make build-linux        - Build for Linux (amd64)"
	@echo "  make build-darwin       - Build for macOS (amd64, arm64)"
	@echo "  make build-windows      - Build for Windows (amd64)"
	@echo ""
	@echo "$(GREEN)Test Commands:$(NC)"
	@echo "  make test               - Run all unit tests"
	@echo "  make test-verbose       - Run tests with verbose output"
	@echo "  make test-race          - Run tests with race detector"
	@echo "  make test-integration   - Run integration tests"
	@echo "  make coverage           - Generate test coverage report"
	@echo "  make coverage-html      - Generate HTML coverage report"
	@echo ""
	@echo "$(GREEN)Code Quality:$(NC)"
	@echo "  make fmt                - Format code with gofmt"
	@echo "  make vet                - Run go vet"
	@echo "  make lint               - Run golangci-lint"
	@echo "  make check              - Run fmt, vet, and lint"
	@echo ""
	@echo "$(GREEN)Vault Commands:$(NC)"
	@echo "  make install            - Install plugin to Vault plugins directory"
	@echo "  make uninstall          - Remove plugin from Vault plugins directory"
	@echo "  make register           - Register plugin with Vault"
	@echo "  make enable             - Enable the secrets engine"
	@echo ""
	@echo "$(GREEN)Utility Commands:$(NC)"
	@echo "  make clean              - Remove build artifacts"
	@echo "  make deps               - Download dependencies"
	@echo "  make tidy               - Tidy and verify go.mod"
	@echo "  make version            - Display version information"
	@echo "  make release-check      - Verify release readiness"
	@echo ""

## all: Build the plugin
all: clean fmt vet test build

## build: Build the plugin for current platform
build:
	@echo "$(BLUE)Building $(BINARY_NAME) for $(shell go env GOOS)/$(shell go env GOARCH)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./main.go
	@echo "$(GREEN)✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

## build-all: Build for all platforms
build-all: build-linux build-darwin build-windows
	@echo "$(GREEN)✓ All platform builds complete$(NC)"

## build-linux: Build for Linux
build-linux:
	@echo "$(BLUE)Building for Linux...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./main.go
	@GOOS=linux GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./main.go
	@echo "$(GREEN)✓ Linux builds complete$(NC)"

## build-darwin: Build for macOS
build-darwin:
	@echo "$(BLUE)Building for macOS...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./main.go
	@GOOS=darwin GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./main.go
	@echo "$(GREEN)✓ macOS builds complete$(NC)"

## build-windows: Build for Windows
build-windows:
	@echo "$(BLUE)Building for Windows...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./main.go
	@echo "$(GREEN)✓ Windows build complete$(NC)"

## clean: Remove build artifacts
clean:
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	@$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)
	@echo "$(GREEN)✓ Clean complete$(NC)"

## test: Run unit tests
test:
	@echo "$(BLUE)Running tests...$(NC)"
	@$(GOTEST) ./... -cover
	@echo "$(GREEN)✓ Tests passed$(NC)"

## test-verbose: Run tests with verbose output
test-verbose:
	@echo "$(BLUE)Running tests (verbose)...$(NC)"
	@$(GOTEST) ./... -v -cover

## test-race: Run tests with race detector
test-race:
	@echo "$(BLUE)Running tests with race detector...$(NC)"
	@$(GOTEST) ./... -race -cover

## test-integration: Run integration tests
test-integration:
	@echo "$(BLUE)Running integration tests...$(NC)"
	@$(GOTEST) ./... -tags=integration -v

## coverage: Generate test coverage report
coverage:
	@echo "$(BLUE)Generating coverage report...$(NC)"
	@$(GOTEST) ./... -coverprofile=$(COVERAGE_FILE)
	@$(GOCMD) tool cover -func=$(COVERAGE_FILE)
	@echo "$(GREEN)✓ Coverage report generated: $(COVERAGE_FILE)$(NC)"

## coverage-html: Generate HTML coverage report
coverage-html: coverage
	@echo "$(BLUE)Generating HTML coverage report...$(NC)"
	@$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "$(GREEN)✓ HTML coverage report: $(COVERAGE_HTML)$(NC)"
	@echo "$(YELLOW)Opening in browser...$(NC)"
	@which xdg-open > /dev/null && xdg-open $(COVERAGE_HTML) || \
	 which open > /dev/null && open $(COVERAGE_HTML) || \
	 echo "Please open $(COVERAGE_HTML) manually"

## fmt: Format Go source code
fmt:
	@echo "$(BLUE)Formatting code...$(NC)"
	@$(GOFMT) ./...
	@echo "$(GREEN)✓ Code formatted$(NC)"

## vet: Run go vet
vet:
	@echo "$(BLUE)Running go vet...$(NC)"
	@$(GOVET) ./...
	@echo "$(GREEN)✓ Vet checks passed$(NC)"

## lint: Run golangci-lint
lint:
	@echo "$(BLUE)Running golangci-lint...$(NC)"
	@which golangci-lint > /dev/null || (echo "$(RED)golangci-lint not installed. Run: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin$(NC)" && exit 1)
	@golangci-lint run ./...
	@echo "$(GREEN)✓ Lint checks passed$(NC)"

## check: Run all code quality checks
check: fmt vet lint
	@echo "$(GREEN)✓ All checks passed$(NC)"

## deps: Download dependencies
deps:
	@echo "$(BLUE)Downloading dependencies...$(NC)"
	@$(GOGET) -v -d ./...
	@echo "$(GREEN)✓ Dependencies downloaded$(NC)"

## tidy: Tidy and verify go.mod
tidy:
	@echo "$(BLUE)Tidying go.mod...$(NC)"
	@$(GOMOD) tidy
	@$(GOMOD) verify
	@echo "$(GREEN)✓ go.mod tidied$(NC)"

## install: Install plugin to Vault plugins directory
install: build
	@echo "$(BLUE)Installing plugin to $(PLUGIN_DIR)...$(NC)"
	@sudo mkdir -p $(PLUGIN_DIR)
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) $(PLUGIN_DIR)/
	@sudo chmod +x $(PLUGIN_DIR)/$(BINARY_NAME)
	@echo "$(GREEN)✓ Plugin installed$(NC)"
	@echo "$(YELLOW)Run 'make register' to register with Vault$(NC)"

## uninstall: Remove plugin from Vault plugins directory
uninstall:
	@echo "$(YELLOW)Removing plugin from $(PLUGIN_DIR)...$(NC)"
	@sudo rm -f $(PLUGIN_DIR)/$(BINARY_NAME)
	@echo "$(GREEN)✓ Plugin removed$(NC)"

## register: Register plugin with Vault
register: install
	@echo "$(BLUE)Registering plugin with Vault...$(NC)"
	@SHA256=$$(sha256sum $(PLUGIN_DIR)/$(BINARY_NAME) | cut -d ' ' -f1) && \
	vault plugin register \
		-sha256=$$SHA256 \
		-command=$(BINARY_NAME) \
		secret skyflow-plugin
	@echo "$(GREEN)✓ Plugin registered$(NC)"
	@echo "$(YELLOW)Run 'make enable' to enable the secrets engine$(NC)"

## enable: Enable the secrets engine
enable:
	@echo "$(BLUE)Enabling skyflow secrets engine...$(NC)"
	@vault secrets enable -path=skyflow skyflow-plugin
	@echo "$(GREEN)✓ Secrets engine enabled at: skyflow/$(NC)"

## version: Display version information
version:
	@echo "$(BLUE)Version Information:$(NC)"
	@echo "  Version: $(VERSION)"
	@echo "  Go Version: $(shell go version)"
	@echo "  Build Time: $(shell date)"
	@echo "  Git Commit: $(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
	@echo "  Git Branch: $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo 'unknown')"

## release-check: Verify release readiness
release-check: clean
	@echo "$(BLUE)Checking release readiness...$(NC)"
	@echo ""
	@echo "$(YELLOW)1. Running tests...$(NC)"
	@$(MAKE) test
	@echo ""
	@echo "$(YELLOW)2. Running code quality checks...$(NC)"
	@$(MAKE) check
	@echo ""
	@echo "$(YELLOW)3. Verifying dependencies...$(NC)"
	@$(MAKE) tidy
	@echo ""
	@echo "$(YELLOW)4. Building all platforms...$(NC)"
	@$(MAKE) build-all
	@echo ""
	@echo "$(YELLOW)5. Checking git status...$(NC)"
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "$(RED)✗ Working directory is not clean$(NC)"; \
		git status --short; \
		exit 1; \
	else \
		echo "$(GREEN)✓ Working directory is clean$(NC)"; \
	fi
	@echo ""
	@echo "$(GREEN)✓ Release checks passed!$(NC)"
	@echo "$(BLUE)Ready to create release tag$(NC)"

## docker-build: Build Docker image
docker-build:
	@echo "$(BLUE)Building Docker image...$(NC)"
	@docker build -t skyflow-plugin:$(VERSION) .
	@echo "$(GREEN)✓ Docker image built: skyflow-plugin:$(VERSION)$(NC)"

## docker-run: Run plugin in Docker container
docker-run:
	@echo "$(BLUE)Running Docker container...$(NC)"
	@docker run --rm -p 8200:8200 skyflow-plugin:$(VERSION)

# Development helpers
.PHONY: watch dev-vault

## watch: Watch for changes and rebuild
watch:
	@echo "$(BLUE)Watching for changes...$(NC)"
	@which fswatch > /dev/null || (echo "$(RED)fswatch not installed$(NC)" && exit 1)
	@fswatch -o $(GO_FILES) | xargs -n1 -I{} make build

## dev-vault: Start Vault in dev mode
dev-vault:
	@echo "$(BLUE)Starting Vault in dev mode...$(NC)"
	@vault server -dev -dev-plugin-dir=./$(BUILD_DIR)

# Benchmarking
.PHONY: bench bench-cache

## bench: Run benchmarks
bench:
	@echo "$(BLUE)Running benchmarks...$(NC)"
	@$(GOTEST) -bench=. -benchmem ./...

## bench-cache: Run cache benchmarks
bench-cache:
	@echo "$(BLUE)Running cache benchmarks...$(NC)"
	@$(GOTEST) -bench=Cache -benchmem ./...

# Generate checksums for releases
.PHONY: checksums

## checksums: Generate SHA256 checksums for binaries
checksums:
	@echo "$(BLUE)Generating checksums...$(NC)"
	@cd $(BUILD_DIR) && sha256sum * > SHA256SUMS
	@echo "$(GREEN)✓ Checksums generated: $(BUILD_DIR)/SHA256SUMS$(NC)"
	@cat $(BUILD_DIR)/SHA256SUMS