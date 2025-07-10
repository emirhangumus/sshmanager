# Variables
BIN_DIR := bin
BIN_NAME := sshmanager
INSTALL_PATH := $(HOME)/.local/bin

# Default target
.DEFAULT_GOAL := help

.PHONY: all build build_compressed install install_compressed run remove clean check_sshpass check_upx help

## Show help
help:
	@echo "Usage:"
	@echo "  make build             - Build the binary"
	@echo "  make build_compressed  - Build and compress the binary with upx"
	@echo "  make install           - Install the binary"
	@echo "  make install_compressed - Install compressed binary"
	@echo "  make remove            - Remove installed binary"
	@echo "  make clean             - Clean build artifacts"

## Build the binary
build:
	@mkdir -p $(BIN_DIR)
	@go build -ldflags="-s -w" -o $(BIN_DIR)/$(BIN_NAME) ./cmd/sshmanager
	@echo "Build complete: $(BIN_DIR)/$(BIN_NAME)"

## Build and compress the binary
build_compressed: check_upx build
	@upx --best --lzma $(BIN_DIR)/$(BIN_NAME)
	@echo "Compressed binary with upx."

## Install the binary
install: check_sshpass build
	@mkdir -p $(INSTALL_PATH)
	@cp $(BIN_DIR)/$(BIN_NAME) $(INSTALL_PATH)
	@echo "Installed to $(INSTALL_PATH)"

## Install the compressed binary
install_compressed: check_sshpass check_upx build_compressed
	@mkdir -p $(INSTALL_PATH)
	@cp $(BIN_DIR)/$(BIN_NAME) $(INSTALL_PATH)
	@echo "Compressed binary installed to $(INSTALL_PATH)"

## Run the binary
run: build
	@$(INSTALL_PATH)/$(BIN_NAME)

## Remove the installed binary
remove:
	@rm -f $(INSTALL_PATH)/$(BIN_NAME)
	@echo "Removed from $(INSTALL_PATH)"

## Clean build artifacts
clean:
	@rm -rf $(BIN_DIR)
	@echo "Cleaned build artifacts."

## Check for sshpass
check_sshpass:
	@command -v sshpass >/dev/null 2>&1 && \
	echo "[✓] sshpass is already installed." || \
	(sudo apt install sshpass -y && \
	echo "[+] Installed sshpass.")

## Check for upx
check_upx:
	@command -v upx >/dev/null 2>&1 && \
	echo "[✓] upx is already installed." || \
	(sudo apt install upx -y && \
	echo "[+] Installed upx.")
