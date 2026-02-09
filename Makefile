# Variables
APP_NAME := sshmanager
BIN_DIR := bin
BIN_NAME := $(APP_NAME)
INSTALL_PATH := $(HOME)/.local/bin
CMD_PATH := ./cmd/$(APP_NAME)

# Build variables
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)

# Go variables
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

.DEFAULT_GOAL := help

.PHONY: all help info build build-compressed install install-compressed run remove clean test test-coverage lint fmt vet mod-tidy mod-download deps release snapshot check-sshpass check-upx check-goreleaser

help:
	@echo "$(APP_NAME) - SSH Manager"
	@echo ""
	@echo "Build Commands:"
	@echo "  make build              - Build the binary"
	@echo "  make build-compressed   - Build and compress with UPX"
	@echo "  make release            - Create release with GoReleaser"
	@echo "  make snapshot           - Create snapshot release"
	@echo ""
	@echo "Install Commands:"
	@echo "  make install            - Install binary to $(INSTALL_PATH)"
	@echo "  make install-compressed - Install compressed binary"
	@echo "  make remove             - Remove installed binary"
	@echo ""
	@echo "Development Commands:"
	@echo "  make run                - Build and run the application"
	@echo "  make test               - Run tests"
	@echo "  make test-coverage      - Run tests with coverage"
	@echo "  make lint               - Run golangci-lint"
	@echo "  make fmt                - Format code with gofmt"
	@echo "  make vet                - Run go vet"
	@echo ""
	@echo "Dependency Commands:"
	@echo "  make deps               - Verify tool dependencies"
	@echo "  make mod-tidy           - Tidy go modules"
	@echo "  make mod-download       - Download go modules"
	@echo ""
	@echo "Utility Commands:"
	@echo "  make clean              - Clean build artifacts"
	@echo "  make info               - Show build information"

info:
	@echo "Build Information:"
	@echo "  App Name:     $(APP_NAME)"
	@echo "  Version:      $(VERSION)"
	@echo "  Commit:       $(COMMIT)"
	@echo "  Build Time:   $(BUILD_TIME)"
	@echo "  GOOS:         $(GOOS)"
	@echo "  GOARCH:       $(GOARCH)"
	@echo "  Install Path: $(INSTALL_PATH)"

build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BIN_DIR)
	@go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BIN_NAME) $(CMD_PATH)
	@echo "✓ Build complete: $(BIN_DIR)/$(BIN_NAME)"

build-compressed: check-upx build
	@echo "Compressing binary with UPX..."
	@upx --best --lzma $(BIN_DIR)/$(BIN_NAME)
	@echo "✓ Compressed binary with UPX"

install: check-sshpass build
	@echo "Installing $(APP_NAME)..."
	@mkdir -p $(INSTALL_PATH)
	@cp $(BIN_DIR)/$(BIN_NAME) $(INSTALL_PATH)
	@chmod +x $(INSTALL_PATH)/$(BIN_NAME)
	@echo "✓ Installed to $(INSTALL_PATH)"
	@echo "Make sure $(INSTALL_PATH) is in your PATH"

install-compressed: check-sshpass build-compressed
	@echo "Installing compressed $(APP_NAME)..."
	@mkdir -p $(INSTALL_PATH)
	@cp $(BIN_DIR)/$(BIN_NAME) $(INSTALL_PATH)
	@chmod +x $(INSTALL_PATH)/$(BIN_NAME)
	@echo "✓ Compressed binary installed to $(INSTALL_PATH)"

run: build
	@echo "Running $(APP_NAME)..."
	@$(BIN_DIR)/$(BIN_NAME)

remove:
	@echo "Removing $(APP_NAME)..."
	@rm -f $(INSTALL_PATH)/$(BIN_NAME)
	@echo "✓ Removed from $(INSTALL_PATH)"

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR) dist/ coverage.out coverage.html
	@go clean -cache -testcache
	@echo "✓ Cleaned build artifacts"

test:
	@echo "Running tests..."
	@go test -v ./...

test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

lint:
	@echo "Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "✓ Code formatted"

vet:
	@echo "Running go vet..."
	@go vet ./...
	@echo "✓ go vet passed"

mod-tidy:
	@echo "Tidying go modules..."
	@go mod tidy
	@echo "✓ Modules tidied"

mod-download:
	@echo "Downloading go modules..."
	@go mod download
	@echo "✓ Modules downloaded"

deps: check-sshpass check-upx check-goreleaser mod-download
	@echo "✓ Dependencies verified"

release: check-goreleaser
	@echo "Creating release with GoReleaser..."
	@goreleaser release --clean
	@echo "✓ Release created"

snapshot: check-goreleaser
	@echo "Creating snapshot release..."
	@goreleaser release --snapshot --clean
	@echo "✓ Snapshot release created"

check-sshpass:
	@command -v sshpass >/dev/null 2>&1 || (echo "sshpass is required but not installed." && exit 1)
	@echo "[✓] sshpass is installed"

check-upx:
	@command -v upx >/dev/null 2>&1 || (echo "upx is required for compressed builds but not installed." && exit 1)
	@echo "[✓] upx is installed"

check-goreleaser:
	@command -v goreleaser >/dev/null 2>&1 || (echo "goreleaser is required for release targets but not installed." && exit 1)
	@echo "[✓] goreleaser is installed"
