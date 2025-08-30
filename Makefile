.PHONY: build run clean test fmt vet deps install dev help

# Variables
BINARY_NAME=twist
MAIN_PACKAGE=.
BUILD_DIR=bin
GO_FILES=$(shell find . -name '*.go' -type f -not -path "./vendor/*")

# Default target
all: build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags="-X main.version=dev-$(shell git rev-parse --short HEAD) -X main.commit=$(shell git rev-parse HEAD) -X main.date=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ')" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Built $(BINARY_NAME) in $(BUILD_DIR)/ directory"

# Build and run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	@./$(BUILD_DIR)/$(BINARY_NAME) login.ts

# Run without building (useful during development)
dev:
	@echo "Running $(BINARY_NAME) in development mode..."
	@go run $(MAIN_PACKAGE) login.ts

# Install dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies updated"

# Format Go code
fmt:
	@echo "Formatting Go code..."
	@go fmt ./...
	@echo "Code formatted"

# Run go vet
vet:
	@echo "Vetting Go code..."
	@go vet ./...
	@echo "Code vetted"

# Run tests (including integration tests)
test:
	@echo "Running all tests..."
	@go test -timeout=5s -tags=integration -v -short ./... -p=1 -json 2>&1 | tee test_output.json | jq -r 'select(.Action == "pass" or .Action == "fail" or .Action == "skip" or .Action == "build-output") | if .Action == "build-output" then ("BUILD ERROR: " + .Output) else (select(.Test) | "\(.Action | ascii_upcase): \(.Test)") end' 2>/dev/null | grep -v "^$$" | grep -E "FAIL|SKIP|BUILD ERROR" || echo "Tests completed"
	@if grep -q '"Action":"fail"' test_output.json 2>/dev/null || grep -q "panic: test timed out" test_output.json 2>/dev/null; then \
		if grep -q "panic: test timed out" test_output.json; then \
			echo "TIMEOUTS:"; \
			jq -r 'select(.Output and (.Output | contains("panic: test timed out"))) | .Test' test_output.json 2>/dev/null | sort -u | sed 's/^/  /' || true; \
		fi; \
		exit 1; \
	fi
	@echo "All tests passed"
	@rm -f test_output.json

# Run integration tests only
test-integration:
	@echo "Running integration tests..."
	@go test -tags=integration -v ./integration/...

# Run scripting engine tests only (unit tests)
test-scripting:
	@echo "Running scripting engine unit tests..."
	@go test -v -short -race ./internal/scripting/... ./internal/scripting/vm/commands/...

# Run tests with coverage
test-coverage-scripting:
	@echo "Running scripting tests with coverage..."
	@go test -v -short -race -coverprofile=coverage-scripting.out ./internal/scripting/... ./internal/scripting/vm/commands/...
	@go tool cover -html=coverage-scripting.out -o coverage-scripting.html
	@echo "Scripting coverage report generated: coverage-scripting.html"

# Run specific test by name
test-run:
	@echo "Usage: make test-run TEST=TestName"
	@if [ -z "$(TEST)" ]; then echo "Please specify TEST=TestName"; exit 1; fi
	@go test -v -short -race -run $(TEST) ./internal/scripting/...

# Benchmark scripting performance  
bench-scripting:
	@echo "Running scripting benchmarks..."
	@go test -bench=. -benchmem ./internal/scripting/...

# Build test harness
build-test-harness:
	@echo "Building test harness..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/test_harness ./cmd/test_harness
	@echo "Test harness built: $(BUILD_DIR)/test_harness"

# Run script tests with harness
test-scripts: build-test-harness
	@echo "Running script tests..."
	@./$(BUILD_DIR)/test_harness -basic

# Test all TWX scripts  
test-all-scripts: build-test-harness
	@echo "Testing all TWX scripts..."
	@./$(BUILD_DIR)/test_harness -all

# Test single script
test-script: build-test-harness
	@echo "Usage: make test-script SCRIPT=path/to/script.twx"
	@if [ -z "$(SCRIPT)" ]; then echo "Please specify SCRIPT=path/to/file"; exit 1; fi
	@./$(BUILD_DIR)/test_harness $(SCRIPT)

# Run tests with coverage
test-coverage:
	@echo "Running unit tests with coverage..."
	@go test -v -short -race -coverprofile=coverage.out ./...
	@echo "Running integration tests with coverage..."
	@go test -tags=integration -v -short -race -coverprofile=coverage-integration.out ./integration/...
	@go tool cover -html=coverage.out -o coverage.html
	@go tool cover -html=coverage-integration.out -o coverage-integration.html
	@echo "Coverage reports generated: coverage.html, coverage-integration.html"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@go clean
	@echo "Clean complete"

# Install the binary to $GOPATH/bin or $GOBIN
install:
	@echo "Installing $(BINARY_NAME)..."
	@go install $(MAIN_PACKAGE)
	@echo "$(BINARY_NAME) installed"

# Check for common issues
check: fmt vet test
	@echo "All checks passed"

# Development build with log flags
build-debug:
	@echo "Building $(BINARY_NAME) with debug flags..."
	@mkdir -p $(BUILD_DIR)
	@go build -gcflags="all=-N -l" -o $(BUILD_DIR)/$(BINARY_NAME)-debug $(MAIN_PACKAGE)
	@echo "Debug build complete: $(BUILD_DIR)/$(BINARY_NAME)-debug"

# Build for multiple platforms using GoReleaser (development snapshots)
build-all:
	@echo "Building for multiple platforms with GoReleaser..."
	@~/go/bin/goreleaser build --snapshot --clean
	@echo "Multi-platform builds complete in dist/ directory"

# Release using GoReleaser (requires git tag)
release:
	@echo "Creating release with GoReleaser..."
	@~/go/bin/goreleaser release --clean

# Check GoReleaser configuration
release-check:
	@echo "Checking GoReleaser configuration..."
	@~/go/bin/goreleaser check

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build the application"
	@echo "  run           - Build and run the application"
	@echo "  dev           - Run in development mode (no build step)"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  fmt           - Format Go code"
	@echo "  vet           - Run go vet"
	@echo "  test          - Run all tests (unit + integration)"
	@echo "  test-integration - Run integration tests only"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  clean         - Clean build artifacts"
	@echo "  install       - Install binary to GOPATH/bin"
	@echo "  check         - Run fmt, vet, and test"
	@echo "  build-debug   - Build with debug flags"
	@echo "  build-all     - Build for multiple platforms using GoReleaser"
	@echo "  release       - Create release with GoReleaser (requires git tag)"
	@echo "  release-check - Check GoReleaser configuration"
	@echo "  build-test-harness - Build the script test harness"  
	@echo "  test-scripts  - Run basic script tests"
	@echo "  test-all-scripts - Test all TWX scripts in twx-scripts/"
	@echo "  test-script   - Test single script (use SCRIPT=path)"
	@echo "  test-scripting - Run scripting engine unit tests"
	@echo "  test-coverage-scripting - Run scripting tests with coverage"
	@echo "  test-run      - Run specific test (use TEST=TestName)"
	@echo "  bench-scripting - Run scripting benchmarks"
	@echo "  help          - Show this help message"
