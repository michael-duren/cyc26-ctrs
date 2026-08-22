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

setup-node: buildc
	@bash scripts/setup.sh $(THEME)

setup-node-cyc: buildc
	@bash scripts/setup.sh cyc

setup-node-nagios: buildc
	@bash scripts/setup.sh nagios

buildc:
	@echo "building breakout"
	@cd scripts && gcc -static -o breakout breakout.c && echo "successfully built breakout"
