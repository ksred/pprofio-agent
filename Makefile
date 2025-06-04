.PHONY: build test lint example clean

# Version
VERSION := 0.1.0

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOVET := $(GOCMD) vet
GOLINT := golangci-lint
GOCOVER := $(GOCMD) tool cover
GOMOD := $(GOCMD) mod

# Files
MAIN_PKG := ./examples/basic
PACKAGES := $(shell $(GOCMD) list ./...)

all: build

# Build the library
build:
	$(GOBUILD) -v $(PACKAGES)

# Build the example
example: 
	$(GOBUILD) -o bin/example $(MAIN_PKG)

# Run tests
test:
	$(GOTEST) -v -race $(PACKAGES)

# Run tests with coverage
cover:
	$(GOTEST) -coverprofile=coverage.out $(PACKAGES)
	$(GOCOVER) -html=coverage.out

# Run linting
lint:
	$(GOLINT) run

# Verify dependencies
verify:
	$(GOMOD) verify

# Tidy go.mod
tidy:
	$(GOMOD) tidy

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out

# Run the example
run-example: example
	./bin/example