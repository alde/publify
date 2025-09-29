# Publify

A CLI tool for converting documents between formats, optimized for e-readers like Kobo, Kindle, and others.

## Features

- **PDF to EPUB conversion** with reader-specific optimizations
- **Metadata editing** for EPUB files
- **Multi-format support** designed for various e-reader devices
- **Optimization profiles** for different reader capabilities

## Installation

### Build from source

```bash
git clone https://github.com/alde/publify.git
cd publify
go build -o publify
```

## Usage

### Basic Commands

```bash
# Convert PDF to EPUB
publify convert input.pdf output.epub

# Edit EPUB metadata
publify metadata input.epub

# Show help
publify --help

# Enable verbose output
publify --verbose convert input.pdf output.epub
```

### Supported Formats

- **Input**: PDF
- **Output**: EPUB

## Project Structure

```
publify/
├── cmd/                 # CLI commands and subcommands
├── internal/           # Internal packages
│   └── worker/        # Worker pool for concurrent processing
├── pkg/               # Public packages
│   ├── converter/     # Format conversion logic
│   ├── metadata/      # Metadata handling
│   ├── progress/      # Progress indicators
│   └── reader/        # E-reader profiles and capabilities
└── testdata/          # Test files and fixtures
```

## Dependencies

- [cobra](https://github.com/spf13/cobra) - CLI framework
- [go-epub](https://github.com/bmaupin/go-epub) - EPUB generation
- [imaging](https://github.com/disintegration/imaging) - Image processing
- [pdf](https://github.com/ledongthuc/pdf) - PDF reading
- [webp](https://github.com/chai2010/webp) - WebP image support

## Requirements

- Go 1.25.0 or later

## Development

### Building

```bash
go build -o publify
```

### Testing

The project includes comprehensive unit and integration tests:

```bash
# Run all tests
make test

# Run only unit tests
make test-unit

# Run only integration tests (requires test PDF files)
make test-integration

# Run tests with verbose output
make test-verbose

# Generate coverage report
make coverage

# Run specific integration test with Romeo and Juliet PDF
make test-romeo
```

**Manual testing:**
```bash
# Unit tests only
go test ./pkg/...

# All tests including integration
go test ./...

# Verbose output
go test -v ./pkg/converter
```

**Test files:** Place PDF test files in the `testdata/` directory. The integration tests will automatically detect and use available PDF files.

## License

This project is licensed under the terms specified in the repository.

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.