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

# Colors for output
GREEN := \033[32m
YELLOW := \033[33m
RED := \033[31m
BLUE := \033[34m
RESET := \033[0m

# Default target
.DEFAULT_GOAL := help

.PHONY: all build build-compressed install install-compressed run remove clean test lint fmt vet mod-tidy mod-download deps release snapshot check-sshpass check-upx check-goreleaser help

## Show help
help:
	@echo "$(BLUE)$(APP_NAME) - SSH Manager$(RESET)"
	@echo ""
	@echo "$(GREEN)Build Commands:$(RESET)"
	@echo "  make build              - Build the binary"
	@echo "  make build-compressed   - Build and compress with UPX"
	@echo "  make release            - Create release with GoReleaser"
	@echo "  make snapshot           - Create snapshot release"
	@echo ""
	@echo "$(GREEN)Install Commands:$(RESET)"
	@echo "  make install            - Install binary to $(INSTALL_PATH)"
	@echo "  make install-compressed - Install compressed binary"
	@echo "  make remove             - Remove installed binary"
	@echo ""
	@echo "$(GREEN)Development Commands:$(RESET)"
	@echo "  make run                - Build and run the application"
	@echo "  make test               - Run tests"
	@echo "  make test-coverage      - Run tests with coverage"
	@echo "  make lint               - Run golangci-lint"
	@echo "  make fmt                - Format code with gofmt"
	@echo "  make vet                - Run go vet"
	@echo ""
	@echo "$(GREEN)Dependency Commands:$(RESET)"
	@echo "  make deps               - Install all dependencies"
	@echo "  make mod-tidy           - Tidy go modules"
	@echo "  make mod-download       - Download go modules"
	@echo ""
	@echo "$(GREEN)Utility Commands:$(RESET)"
	@echo "  make clean              - Clean build artifacts"
	@echo "  make info               - Show build information"

## Build information
info:
	@echo "$(BLUE)Build Information:$(RESET)"
	@echo "  App Name:    $(APP_NAME)"
	@echo "  Version:     $(VERSION)"
	@echo "  Commit:      $(COMMIT)"
	@echo "  Build Time:  $(BUILD_TIME)"
	@echo "  GOOS:        $(GOOS)"
	@echo "  GOARCH:      $(GOARCH)"
	@echo "  Install Path: $(INSTALL_PATH)"

## Build the binary
build:
	@echo "$(GREEN)Building $(APP_NAME)...$(RESET)"
	@mkdir -p $(BIN_DIR)
	@go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BIN_NAME) $(CMD_PATH)
	@echo "$(GREEN)✓ Build complete: $(BIN_DIR)/$(BIN_NAME)$(RESET)"

## Build and compress the binary
build-compressed: check-upx build
	@echo "$(GREEN)Compressing binary with UPX...$(RESET)"
	@upx --best --lzma $(BIN_DIR)/$(BIN_NAME)
	@echo "$(GREEN)✓ Compressed binary with UPX$(RESET)"

## Install the binary
install: check-sshpass build
	@echo "$(GREEN)Installing $(APP_NAME)...$(RESET)"
	@mkdir -p $(INSTALL_PATH)
	@cp $(BIN_DIR)/$(BIN_NAME) $(INSTALL_PATH)
	@chmod +x $(INSTALL_PATH)/$(BIN_NAME)
	@echo "$(GREEN)✓ Installed to $(INSTALL_PATH)$(RESET)"
	@echo "$(YELLOW)Make sure $(INSTALL_PATH) is in your PATH$(RESET)"

## Install the compressed binary
install-compressed: check-sshpass build-compressed
	@echo "$(GREEN)Installing compressed $(APP_NAME)...$(RESET)"
	@mkdir -p $(INSTALL_PATH)
	@cp $(BIN_DIR)/$(BIN_NAME) $(INSTALL_PATH)
	@chmod +x $(INSTALL_PATH)/$(BIN_NAME)
	@echo "$(GREEN)✓ Compressed binary installed to $(INSTALL_PATH)$(RESET)"

## Run the binary
run: build
	@echo "$(GREEN)Running $(APP_NAME)...$(RESET)"
	@$(BIN_DIR)/$(BIN_NAME)

## Remove the installed binary
remove:
	@echo "$(YELLOW)Removing $(APP_NAME)...$(RESET)"
	@rm -f $(INSTALL_PATH)/$(BIN_NAME)
	@echo "$(GREEN)✓ Removed from $(INSTALL_PATH)$(RESET)"

## Clean build artifacts
clean:
	@echo "$(YELLOW)Cleaning build artifacts...$(RESET)"
	@rm -rf $(BIN_DIR) dist/
	@go clean -cache -testcache -modcache
	@echo "$(GREEN)✓ Cleaned build artifacts$(RESET)"

## Run tests
test:
	@echo "$(GREEN)Running tests...$(RESET)"
	@go test -v ./...

## Run tests with coverage
test-coverage:
	@echo "$(GREEN)Running tests with coverage...$(RESET)"
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✓ Coverage report generated: coverage.html$(RESET)"

## Run golangci-lint
lint:
	@echo "$(GREEN)Running golangci-lint...$(RESET)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
	    golangci-lint run; \
	else \
	    echo "$(RED)golangci-lint not found. Install it with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest$(RESET)"; \
	fi

## Format code
fmt:
	@echo "$(GREEN)Formatting code...$(RESET)"
	@go fmt ./...
	@echo "$(GREEN)✓ Code formatted$(RESET)"

## Run go vet
vet:
	@echo "$(GREEN)Running go vet...$(RESET)"
	@go vet ./...
	@echo "$(GREEN)✓ go vet passed$(RESET)"

## Tidy go modules
mod-tidy:
	@echo "$(GREEN)Tidying go modules...$(RESET)"
	@go mod tidy
	@echo "$(GREEN)✓ Modules tidied$(RESET)"

## Download go modules
mod-download:
	@echo "$(GREEN)Downloading go modules...$(RESET)"
	@go mod download
	@echo "$(GREEN)✓ Modules downloaded$(RESET)"

## Install all dependencies
deps: check-sshpass check-upx check-goreleaser mod-download
	@echo "$(GREEN)✓ All dependencies checked/installed$(RESET)"

## Create release with GoReleaser
release: check-goreleaser
	@echo "$(GREEN)Creating release with GoReleaser...$(RESET)"
	@goreleaser release --clean
	@echo "$(GREEN)✓ Release created$(RESET)"

## Create snapshot release
snapshot: check-goreleaser
	@echo "$(GREEN)Creating snapshot release...$(RESET)"
	@goreleaser release --snapshot --clean
	@echo "$(GREEN)✓ Snapshot release created$(RESET)"

## Check for sshpass
check-sshpass:
	@command -v sshpass >/dev/null 2>&1 && \
	echo "$(GREEN)[✓] sshpass is installed$(RESET)" || \
	(echo "$(YELLOW)[!] Installing sshpass...$(RESET)" && \
	sudo apt update && sudo apt install sshpass -y && \
	echo "$(GREEN)[✓] sshpass installed$(RESET)")

## Check for upx
check-upx:
	@command -v upx >/dev/null 2>&1 && \
	echo "$(GREEN)[✓] upx is installed$(RESET)" || \
	(echo "$(YELLOW)[!] Installing upx...$(RESET)" && \
	sudo apt update && sudo apt install upx -y && \
	echo "$(GREEN)[✓] upx installed$(RESET)")

## Check for goreleaser
check-goreleaser:
	@command -v goreleaser >/dev/null 2>&1 && \
	echo "$(GREEN)[✓] goreleaser is installed$(RESET)" || \
	(echo "$(YELLOW)[!] Installing goreleaser...$(RESET)" && \
	echo 'deb [trusted=yes] https://repo.goreleaser.com/apt/ /' | sudo tee /etc/apt/sources.list.d/goreleaser.list && \
	sudo apt update && sudo apt install goreleaser -y && \
	echo "$(GREEN)[✓] goreleaser installed$(RESET)")