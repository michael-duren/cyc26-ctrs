BINARY := demo
PKG := ./cmd/demo
BIN_DIR := bin
PREFIX ?= $(HOME)/.local

# webapp branding: cyc (Commit Your Code) or nagios
THEME ?= cyc

.PHONY: run build setup-node setup-node-cyc setup-node-nagios lint

run:
	@go run $(PKG) run "/bin/bash"

sudo:
	sudo go run $(PKG) run "/bin/bash"

reexec:
	@echo "warning: this is for testing"
	@echo "this should only be called internally"
	@go run $(PKG) reexec "/bin/bash" "node"

build:
	@go build -o $(BINARY) $(PKG)

lint:
	@golangci-lint run ./...

setup-node:
	@bash scripts/setup.sh $(THEME)

setup-node-cyc:
	@bash scripts/setup.sh cyc

setup-node-nagios:
	@bash scripts/setup.sh nagios
