# Publify

A CLI tool for converting documents between formats, optimized for e-readers like Kobo, Kindle, and others.

## Features

- **PDF to EPUB conversion** with reader-specific optimizations
- **Metadata editing** for EPUB files
- **EPUB extraction and compression** for manual editing workflows
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
publify convert input.pdf -o output.epub

# Edit EPUB metadata
publify metadata book.epub --title "New Title" --author "Author Name"

# Extract EPUB for manual editing
publify extract book.epub -o extracted_folder/

# Compress folder back to EPUB
publify compress extracted_folder/ -o modified_book.epub

# Show help
publify --help

# Enable verbose output
publify --verbose convert input.pdf -o output.epub
```

### Manual EPUB Editing Workflow

For complex EPUB modifications that require manual editing:

```bash
# 1. Extract EPUB to a folder
publify extract book.epub -o book_folder/

# 2. Edit files manually in book_folder/
#    - Modify HTML files in OEBPS/
#    - Update CSS styles
#    - Replace images
#    - Edit metadata in content.opf

# 3. Compress back to EPUB
publify compress book_folder/ -o fixed_book.epub
```

The extracted folder maintains the standard EPUB structure:
```
book_folder/
├── mimetype
├── META-INF/
│   └── container.xml
└── OEBPS/
    ├── content.opf     # Package metadata
    ├── toc.ncx         # Table of contents
    ├── styles/         # CSS files
    ├── images/         # Image assets
    └── text/           # HTML content files
```

### Supported Formats

- **Input**: PDF (for conversion), EPUB (for extraction/metadata editing)
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

## Key Dependencies

- [cobra](https://github.com/spf13/cobra) - CLI framework
- [go-epub](https://github.com/bmaupin/go-epub) - EPUB generation
- [imaging](https://github.com/disintegration/imaging) - Image processing
- [go-pdfium](https://github.com/klippa-app/go-pdfium) - PDF processing
- [webp](https://github.com/chai2010/webp) - WebP image support
- [humanize](https://github.com/dustin/go-humanize) - Human-readable formatting

## Requirements

- Go 1.25.0 or later

## Development

### Building

```bash
go build -o publify
```

### Code Quality

This project follows strict formatting and quality standards:

```bash
# Setup pre-commit hooks for automatic formatting
make setup-hooks

# Format code manually
make fmt

# Run static analysis
make lint

# Run all quality checks
make check
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
