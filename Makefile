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
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Built $(BINARY_NAME) in $(BUILD_DIR)/ directory"

# Build and run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	@./$(BUILD_DIR)/$(BINARY_NAME)

# Run without building (useful during development)
dev:
	@echo "Running $(BINARY_NAME) in development mode..."
	@go run $(MAIN_PACKAGE)

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

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

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

# Development build with debug flags
build-debug:
	@echo "Building $(BINARY_NAME) with debug flags..."
	@mkdir -p $(BUILD_DIR)
	@go build -gcflags="all=-N -l" -o $(BUILD_DIR)/$(BINARY_NAME)-debug $(MAIN_PACKAGE)
	@echo "Debug build complete: $(BUILD_DIR)/$(BINARY_NAME)-debug"

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)
	@GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
	@GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PACKAGE)
	@GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)
	@echo "Multi-platform builds complete"

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build the application"
	@echo "  run           - Build and run the application"
	@echo "  dev           - Run in development mode (no build step)"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  fmt           - Format Go code"
	@echo "  vet           - Run go vet"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  clean         - Clean build artifacts"
	@echo "  install       - Install binary to GOPATH/bin"
	@echo "  check         - Run fmt, vet, and test"
	@echo "  build-debug   - Build with debug flags"
	@echo "  build-all     - Build for multiple platforms"
	@echo "  help          - Show this help message"