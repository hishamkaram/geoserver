## Makefile for github.com/hishamkaram/geoserver
##
## Common targets:
##   make test             # unit + integration
##   make test-unit        # fast, no Docker
##   make test-integration # requires `make compose-up`
##   make lint             # golangci-lint
##   make fmt              # format with gofmt + goimports
##   make tidy             # go mod tidy && verify
##   make vuln             # govulncheck
##   make cover            # unit tests with coverage profile
##   make compose-up       # boot the dev GeoServer + PostGIS stack
##   make compose-down     # tear it down

SHELL := /usr/bin/env bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := help

GO          ?= go
GOBIN       ?= $(shell $(GO) env GOPATH)/bin
COMPOSE     ?= docker compose
COMPOSE_FILE ?= docker-compose.yml
PKG         := ./...
COVER_FILE  := coverage.out
INT_TAG     := integration

GOLANGCI_LINT_VERSION ?= v2.12.1

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "Targets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: tidy
tidy: ## go mod tidy && verify
	$(GO) mod tidy
	$(GO) mod verify

.PHONY: build
build: ## go build all packages
	$(GO) build $(PKG)

.PHONY: vet
vet: ## go vet
	$(GO) vet $(PKG)

.PHONY: fmt
fmt: ## gofmt + goimports
	gofmt -s -w .
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w -local github.com/hishamkaram/geoserver .; \
	else \
		echo "goimports not installed; skipping. Install: go install golang.org/x/tools/cmd/goimports@latest"; \
	fi

.PHONY: lint
lint: ## golangci-lint run (installs if missing)
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."; \
		$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
	fi
	golangci-lint run $(PKG)

.PHONY: vuln
vuln: ## govulncheck (installs if missing)
	@if ! command -v govulncheck >/dev/null 2>&1; then \
		$(GO) install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi
	govulncheck $(PKG)

.PHONY: test
test: test-unit test-integration ## unit + integration tests

.PHONY: test-unit
test-unit: ## unit tests (no Docker)
	$(GO) test -race -short -timeout=60s $(PKG)

.PHONY: test-integration
test-integration: ## integration tests against compose stack
	$(GO) test -race -tags=$(INT_TAG) -timeout=10m $(PKG)

.PHONY: cover
cover: ## unit tests with coverage profile
	$(GO) test -race -short -coverprofile=$(COVER_FILE) -covermode=atomic $(PKG)
	$(GO) tool cover -func=$(COVER_FILE) | tail -n 1

.PHONY: cover-html
cover-html: cover ## render coverage HTML report
	$(GO) tool cover -html=$(COVER_FILE)

.PHONY: compose-up
compose-up: ## boot dev GeoServer + PostGIS
	$(COMPOSE) -f $(COMPOSE_FILE) up -d --wait

.PHONY: compose-down
compose-down: ## tear down dev stack
	$(COMPOSE) -f $(COMPOSE_FILE) down -v

.PHONY: compose-logs
compose-logs: ## tail compose logs
	$(COMPOSE) -f $(COMPOSE_FILE) logs -f

.PHONY: compose-test-up
compose-test-up: ## boot the integration-test compose (2.27 LTS leg)
	$(COMPOSE) -f docker-compose.test.yml up -d --wait

.PHONY: compose-test-down
compose-test-down: ## tear down integration-test compose
	$(COMPOSE) -f docker-compose.test.yml down -v

.PHONY: clean
clean: ## remove generated artifacts
	rm -f $(COVER_FILE)
	$(GO) clean -testcache
