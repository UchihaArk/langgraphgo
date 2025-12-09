# Makefile for LangGraphGo Chat Demo

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary names
BINARY_NAME=chat
BINARY_UNIX=$(BINARY_NAME)_unix

# Build flags
LDFLAGS=-ldflags "-X main.version=$(shell git describe --tags --always --dirty 2>/dev/null || echo 'dev')"

.PHONY: all build clean test coverage deps help run docker-build docker-run format lint install

# Main targets
all: build

# Build the application
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) -v .

# Build for Linux
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_UNIX) -v .

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f coverage.out

# Run tests
test:
	$(GOTEST) -v ./...

# Run tests with coverage
coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Format code
format:
	$(GOCMD) fmt ./...

# Lint code (requires golangci-lint)
lint:
	golangci-lint run

# Install the application
install: build
	cp $(BINARY_NAME) $(GOPATH)/bin/

# Run the application
run:
	$(GOBUILD) -o $(BINARY_NAME) -v .
	./$(BINARY_NAME)

# Run with environment file
run-env:
	$(GOBUILD) -o $(BINARY_NAME) -v .
	./$(BINARY_NAME) --env .env

# Development mode with hot reload (requires air)
dev:
	air -c .air.toml

# Docker build
docker-build:
	docker build -t langgraphgo-chat:latest .

# Docker run
docker-run:
	docker run -p 8080:8080 --env-file .env langgraphgo-chat:latest

# Production build
prod-build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -a -installsuffix cgo $(LDFLAGS) -o $(BINARY_UNIX) .

# Check for required tools
check-tools:
	@command -v go >/dev/null 2>&1 || { echo "Go is required but not installed. Aborting." >&2; exit 1; }
	@command -v git >/dev/null 2>&1 || { echo "Git is required but not installed. Aborting." >&2; exit 1; }

# Create environment file template
init-env:
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "Created .env file from .env.example"; \
		echo "Please edit .env file with your configuration"; \
	else \
		echo ".env file already exists"; \
	fi

# Create session directory
init-dirs:
	@mkdir -p sessions/clients
	@echo "Created session directories"

# Setup development environment
setup: check-tools init-env init-dirs deps
	@echo "Development environment setup complete"

# Generate documentation
docs:
	godoc -http=:6060

# Pre-commit hook
pre-commit: format lint test
	@echo "Pre-commit checks passed"

# Watch for changes and rebuild (requires fswatch)
watch:
	fswatch -o . | xargs -n1 -I {} make build

# Install development tools
install-tools:
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securecodewarrior/golangci-gosec/v2/cmd/gosec@latest

# Security scan
security:
	gosec ./...

# Benchmark
bench:
	$(GOTEST) -bench=. -benchmem

# Update dependencies
update-deps:
	$(GOGET) -u ./...
	$(GOMOD) tidy

# Verify dependencies
verify-deps:
	$(GOMOD) verify

# Build with race detector
build-race:
	$(GOBUILD) -race $(LDFLAGS) -o $(BINARY_NAME)_race -v .

# Run with race detector
run-race: build-race
	./$(BINARY_NAME)_race

# Profile with pprof
profile:
	$(GOBUILD) -o $(BINARY_NAME)_prof -v .
	./$(BINARY_NAME)_prof -cpuprofile=cpu.prof -memprofile=mem.prof

# Analyze profile
analyze-profile:
	$(GOCMD) tool pprof cpu.prof
	$(GOCMD) tool pprof mem.prof

# Help target
help:
	@echo "Available targets:"
	@echo "  build           - Build the application (with MCP support if available)"
	@echo "  build-linux     - Build for Linux (with MCP support if available)"
	@echo "  clean           - Clean build artifacts"
	@echo "  test            - Run tests"
	@echo "  coverage        - Run tests with coverage"
	@echo "  deps            - Download dependencies"
	@echo "  format          - Format code"
	@echo "  lint            - Lint code"
	@echo "  install         - Install the application"
	@echo "  run             - Run the application"
	@echo "  dev             - Development mode with hot reload"
	@echo "  docker-build    - Build Docker image"
	@echo "  docker-run      - Run Docker container"
	@echo "  setup           - Setup development environment"
	@echo "  help            - Show this help message"
	@echo ""
	@echo "MCP Configuration:"
	@echo "  Set MCP_CONFIG_PATH environment variable to specify MCP config file"
	@echo "  Default: testdata/mcp/mcp.json"