# Publify Makefile

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=publify

# Build targets
.PHONY: all build clean test test-verbose test-unit test-integration coverage help install-deps install-tesseract check-tesseract

all: test build

build: check-tesseract
	$(GOBUILD) -o $(BINARY_NAME) -v .

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

test: test-unit test-integration

test-verbose:
	$(GOTEST) -v ./...

test-unit:
	$(GOTEST) ./pkg/...

test-integration:
	$(GOTEST) -v ./integration_test.go

coverage:
	$(GOTEST) -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Install system dependencies for OCR
install-tesseract:
	@echo "Installing Tesseract OCR dependencies..."
	@if command -v brew >/dev/null 2>&1; then \
		echo "Using Homebrew to install Tesseract..."; \
		brew install tesseract leptonica; \
	elif command -v apt-get >/dev/null 2>&1; then \
		echo "Using apt to install Tesseract..."; \
		sudo apt-get update && sudo apt-get install -y libtesseract-dev libleptonica-dev tesseract-ocr; \
	elif command -v yum >/dev/null 2>&1; then \
		echo "Using yum to install Tesseract..."; \
		sudo yum install -y tesseract-devel leptonica-devel tesseract; \
	elif command -v dnf >/dev/null 2>&1; then \
		echo "Using dnf to install Tesseract..."; \
		sudo dnf install -y tesseract-devel leptonica-devel tesseract; \
	elif command -v pacman >/dev/null 2>&1; then \
		echo "Using pacman to install Tesseract..."; \
		sudo pacman -S tesseract leptonica; \
	else \
		echo "Package manager not detected. Please install Tesseract and Leptonica manually:"; \
		echo "  - Ubuntu/Debian: sudo apt-get install libtesseract-dev libleptonica-dev tesseract-ocr"; \
		echo "  - macOS: brew install tesseract leptonica"; \
		echo "  - RHEL/CentOS: sudo yum install tesseract-devel leptonica-devel"; \
		echo "  - Arch: sudo pacman -S tesseract leptonica"; \
		exit 1; \
	fi
	@echo "Tesseract installation complete!"

check-tesseract:
	@echo "Checking Tesseract installation..."
	@if pkg-config --exists tesseract lept; then \
		echo "✓ Tesseract and Leptonica found"; \
		tesseract --version; \
	else \
		echo "✗ Tesseract or Leptonica not found. Run 'make install-tesseract' first."; \
		exit 1; \
	fi

# Development targets
dev-build: deps install-tesseract test build

install: build
	cp $(BINARY_NAME) /usr/local/bin/

# Test with specific test file
test-romeo:
	$(GOTEST) -v -run TestIntegrationRomeoAndJuliet ./integration_test.go

# Run tests with race detection
test-race:
	$(GOTEST) -race ./...

# Benchmark tests
benchmark:
	$(GOTEST) -bench=. ./...

# Check for Go module issues
check:
	$(GOMOD) verify
	$(GOCMD) vet ./...

# Format code (keeping things lagom and tidy)
fmt:
	$(GOCMD) fmt ./...

# Run static analysis (because code should be clean like a Swedish home)
lint:
	golangci-lint run

# Setup pre-commit hooks (preventing messy commits with Swedish efficiency)
setup-hooks:
	@echo "Setting up pre-commit hooks..."
	@command -v pre-commit >/dev/null 2>&1 || { echo "pre-commit not found. Install with: pip install pre-commit"; exit 1; }
	pre-commit install
	@echo "Pre-commit hooks installed successfully!"

# Run pre-commit on all files
pre-commit-all:
	pre-commit run --all-files

help:
	@echo "Available targets:"
	@echo "  all           - Run tests and build (with OCR support)"
	@echo "  build         - Build the binary (requires Tesseract)"
	@echo "  clean         - Clean build artifacts"
	@echo "  test          - Run all tests"
	@echo "  test-unit     - Run unit tests only"
	@echo "  test-integration - Run integration tests only"
	@echo "  test-verbose  - Run tests with verbose output"
	@echo "  test-race     - Run tests with race detection"
	@echo "  test-romeo    - Run Romeo and Juliet integration test"
	@echo "  coverage      - Generate test coverage report"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  dev-build     - Full development build (installs Tesseract + deps + test + build)"
	@echo "  install       - Install binary to /usr/local/bin"
	@echo "  install-tesseract - Install Tesseract OCR dependencies"
	@echo "  check-tesseract   - Check if Tesseract is properly installed"
	@echo "  benchmark     - Run benchmark tests"
	@echo "  check         - Verify modules and run vet"
	@echo "  fmt           - Format Go code"
	@echo "  lint          - Run static analysis (requires golangci-lint)"
	@echo "  setup-hooks   - Setup pre-commit hooks for formatting enforcement"
	@echo "  pre-commit-all - Run pre-commit hooks on all files"
	@echo "  help          - Show this help message"
