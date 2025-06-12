# Makefile for pprofio Go package

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Build parameters
BINARY_NAME=pprofio
BINARY_UNIX=$(BINARY_NAME)_unix

# Test parameters
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

# Default target
.PHONY: all
all: clean deps test build

# Help
.PHONY: help
help: ## Display available commands
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Development
.PHONY: deps
deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) verify

.PHONY: deps-upgrade
deps-upgrade: ## Upgrade dependencies
	$(GOGET) -u ./...
	$(GOMOD) tidy

.PHONY: tidy
tidy: ## Tidy up go.mod and go.sum
	$(GOMOD) tidy

# Building
.PHONY: build
build: ## Build the package
	$(GOBUILD) -v ./...

.PHONY: clean
clean: ## Remove build artifacts
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f $(COVERAGE_FILE)
	rm -f $(COVERAGE_HTML)

# Testing
.PHONY: test
test: ## Run tests
	$(GOTEST) -v -race ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	$(GOTEST) -v -race -coverprofile=$(COVERAGE_FILE) ./...
	$(GOCMD) tool cover -func=$(COVERAGE_FILE)

.PHONY: test-coverage-html
test-coverage-html: test-coverage ## Generate HTML coverage report
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

.PHONY: test-short
test-short: ## Run short tests only
	$(GOTEST) -short -v ./...

.PHONY: test-stdout
test-stdout: ## Run stdout functionality tests
	$(GOTEST) -v -run ".*[Ss]tdout.*" ./...

.PHONY: benchmark
benchmark: ## Run benchmarks
	$(GOTEST) -bench=. -benchmem ./...

# Code quality
.PHONY: fmt
fmt: ## Format code
	$(GOFMT) -w .

.PHONY: fmt-check
fmt-check: ## Check if code is formatted
	@test -z "$$($(GOFMT) -l .)" || (echo "Code not formatted, run 'make fmt'" && exit 1)

.PHONY: imports
imports: ## Organize imports
	@which goimports > /dev/null || (echo "Installing goimports..." && $(GOGET) golang.org/x/tools/cmd/goimports)
	goimports -w .

.PHONY: imports-check
imports-check: ## Check if imports are organized
	@which goimports > /dev/null || (echo "Installing goimports..." && $(GOGET) golang.org/x/tools/cmd/goimports)
	@test -z "$$(goimports -l .)" || (echo "Imports not organized, run 'make imports'" && exit 1)

.PHONY: lint
lint: ## Run linter
	@which $(GOLINT) > /dev/null || (echo "Installing golangci-lint..." && $(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	$(GOLINT) run --timeout=10m

.PHONY: lint-fix
lint-fix: ## Run linter with auto-fix
	@which $(GOLINT) > /dev/null || (echo "Installing golangci-lint..." && $(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	$(GOLINT) run --fix --timeout=10m

.PHONY: security
security: ## Run security scanner
	@which gosec > /dev/null || (echo "Installing gosec..." && $(GOGET) github.com/securecodewarrior/gosec/v2/cmd/gosec@latest)
	gosec ./...

# Pre-commit checks
.PHONY: pre-commit
pre-commit: fmt-check imports-check lint test ## Run all pre-commit checks

.PHONY: fix
fix: fmt imports lint-fix ## Auto-fix formatting and linting issues

# Release
.PHONY: release
release: ## Create a release (requires VERSION environment variable)
	@if [ -z "$(VERSION)" ]; then echo "VERSION environment variable is required"; exit 1; fi
	@echo "Creating release $(VERSION)..."
	./scripts/release.sh $(VERSION)

.PHONY: release-dry-run
release-dry-run: ## Dry run of release process
	@echo "Performing release dry run..."
	@echo "Running tests..."
	$(MAKE) test
	@echo "Running linting..."
	$(MAKE) lint
	@echo "Checking if working directory is clean..."
	@git status --porcelain | grep -q . && echo "Working directory is not clean" && exit 1 || echo "Working directory is clean"
	@echo "Dry run completed successfully"

# CI/CD simulation
.PHONY: ci
ci: deps test-coverage lint security ## Simulate CI pipeline locally
	@echo "CI pipeline completed successfully"

# Documentation
.PHONY: docs
docs: ## Generate and serve documentation
	@echo "Opening package documentation..."
	@echo "Visit: https://pkg.go.dev/github.com/pprofio/pprofio"

.PHONY: examples
examples: ## Run examples
	@echo "Running examples..."
	$(GOTEST) -v ./examples/...

.PHONY: run-basic-example
run-basic-example: ## Build and run the basic example for demonstration
	@echo "Building and running basic example..."
	@echo "This demonstrates the basic usage of the pprofio package:"
	@echo "----------------------------------------"
	@cd examples/basic && go run main.go
	@echo "----------------------------------------"
	@echo "Example demonstration completed!"

.PHONY: run-stdout-example
run-stdout-example: ## Build and run the stdout example for demonstration
	@echo "Building and running stdout example..."
	@echo "This demonstrates the stdout output functionality:"
	@echo "----------------------------------------"
	@cd examples/stdout && go run main.go -demo
	@echo "----------------------------------------"
	@echo "Example demonstration completed!"

# Development helpers
.PHONY: install-tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	$(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GOGET) golang.org/x/tools/cmd/goimports@latest
	$(GOGET) github.com/securecodewarrior/gosec/v2/cmd/gosec@latest

.PHONY: mod-vendor
mod-vendor: ## Create vendor directory
	$(GOMOD) vendor

.PHONY: mod-graph
mod-graph: ## Show module dependency graph
	$(GOMOD) graph

.PHONY: mod-why
mod-why: ## Explain why packages are needed
	@if [ -z "$(PKG)" ]; then echo "PKG environment variable is required (e.g., make mod-why PKG=github.com/example/pkg)"; exit 1; fi
	$(GOMOD) why $(PKG)

# Git helpers
.PHONY: git-status
git-status: ## Show git status
	@git status --porcelain

.PHONY: git-clean-check
git-clean-check: ## Check if git working directory is clean
	@git status --porcelain | grep -q . && echo "Working directory is not clean" && exit 1 || echo "Working directory is clean"

# Quick development workflow
.PHONY: dev
dev: deps fmt imports test ## Quick development cycle: deps, format, test

.PHONY: quick
quick: fmt test-short ## Quick check: format and short tests

# Performance profiling
.PHONY: profile-cpu
profile-cpu: ## Run CPU profiling
	$(GOTEST) -cpuprofile=cpu.prof -bench=. ./...
	@echo "CPU profile saved to cpu.prof"

.PHONY: profile-mem
profile-mem: ## Run memory profiling
	$(GOTEST) -memprofile=mem.prof -bench=. ./...
	@echo "Memory profile saved to mem.prof"