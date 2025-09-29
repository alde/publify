package converter

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/alde/publify/internal/worker"
	"github.com/alde/publify/pkg/reader"
)

// Options contains conversion settings
type Options struct {
	InputPath      string
	OutputPath     string
	Profile        reader.Profile
	WorkerCount    int
	Verbose        bool
	EnableOCR      bool
	OCRLanguage    string
	ImagePageRange string
}

// Converter handles the PDF to EPUB conversion process
type Converter struct {
	options   Options
	pdfProc   *PDFProcessor
	epubGen   *EPUBGenerator
	stats     ConversionStats
	startTime time.Time
}

// ConversionStats tracks conversion metrics
type ConversionStats struct {
	InputFileSize    uint64
	OutputFileSize   uint64
	PageCount        int
	ProcessedPages   int
	ChapterCount     int
	TextCharCount    int
	ImageCount       int
	ProcessingTime   time.Duration
	CompressionRatio float64
}

// New creates a new converter instance
func New(opts Options) *Converter {
	return &Converter{
		options:   opts,
		startTime: time.Now(),
	}
}

// Convert performs the PDF to EPUB conversion
func (c *Converter) Convert() error {
	ctx := context.Background()

	// Initialize components
	if err := c.initialize(); err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}
	defer c.cleanup()

	// Get input file size for statistics
	inputSize, err := c.pdfProc.GetFileSize()
	if err != nil {
		return fmt.Errorf("failed to get input file size: %w", err)
	}
	c.stats.InputFileSize = uint64(inputSize)

	// Create worker pool with progress tracking
	pool := worker.NewPoolWithProgress(c.options.WorkerCount, c.pdfProc.GetPageCount())
	pool.Start()
	defer pool.Stop()

	if c.options.Verbose {
		fmt.Printf("Starting conversion of %s to %s\n", c.options.InputPath, c.options.OutputPath)
		fmt.Printf("Target reader: %s (%s)\n", c.options.Profile.Name, c.options.Profile.Manufacturer)
		fmt.Printf("Using %d worker goroutines\n", pool.WorkerCount())
	}

	// Process PDF pages
	pages, err := c.pdfProc.ProcessPages(ctx, pool, nil) // Progress handled by worker pool now
	if err != nil {
		return fmt.Errorf("PDF processing failed: %w", err)
	}

	c.stats.PageCount = len(pages)
	c.stats.ProcessedPages = len(pages)

	if c.options.Verbose {
		fmt.Printf("\nProcessed %d pages\n", len(pages))
	}

	// Generate EPUB content
	if err := c.generateEPUB(pages); err != nil {
		return fmt.Errorf("EPUB generation failed: %w", err)
	}

	// Write EPUB file
	if err := c.epubGen.Write(c.options.OutputPath); err != nil {
		return fmt.Errorf("failed to write EPUB: %w", err)
	}

	// Calculate final statistics
	if err := c.calculateFinalStats(); err != nil {
		return fmt.Errorf("failed to calculate final statistics: %w", err)
	}

	// Display results
	c.displayResults()

	return nil
}

// initialize sets up the converter components
func (c *Converter) initialize() error {
	// Initialize PDF processor with image page ranges and OCR options
	pdfProc, err := NewPDFProcessor(c.options.InputPath, c.options.ImagePageRange, c.options.EnableOCR, c.options.OCRLanguage)
	if err != nil {
		return fmt.Errorf("failed to create PDF processor: %w", err)
	}
	c.pdfProc = pdfProc

	// Create EPUB options from input file
	epubOpts := c.createEPUBOptions()

	// Initialize EPUB generator
	c.epubGen = NewEPUBGenerator(c.options.Profile, epubOpts)

	return nil
}

// createEPUBOptions creates EPUB options from the input file
func (c *Converter) createEPUBOptions() EPUBOptions {
	inputName := filepath.Base(c.options.InputPath)
	title := strings.TrimSuffix(inputName, filepath.Ext(inputName))

	return EPUBOptions{
		Title:       title,
		Author:      "Unknown Author",
		Language:    "en",
		Identifier:  fmt.Sprintf("publify-%d", time.Now().Unix()),
		Description: fmt.Sprintf("Converted from %s by Publify", inputName),
	}
}

// progressCallback handles progress updates during PDF processing
func (c *Converter) progressCallback(processed, total int) {
	if c.options.Verbose {
		percentage := float64(processed) / float64(total) * 100
		fmt.Printf("\rProcessing pages: %d/%d (%.1f%%)", processed, total, percentage)
	}
}

// generateEPUB creates the EPUB content from processed pages
func (c *Converter) generateEPUB(pages []PDFPage) error {
	if len(pages) == 0 {
		return fmt.Errorf("no pages to convert")
	}

	// Group pages into chapters (for now, each page is a chapter)
	for _, page := range pages {
		if err := c.epubGen.AddPage(page); err != nil {
			return fmt.Errorf("failed to add page %d: %w", page.Number, err)
		}

		// Update statistics
		c.stats.TextCharCount += len(page.Text)
		if page.HasText {
			c.stats.ChapterCount++
		}
	}

	// Validate EPUB before writing
	if err := c.epubGen.Validate(); err != nil {
		return fmt.Errorf("EPUB validation failed: %w", err)
	}

	return nil
}

// calculateFinalStats computes final conversion statistics
func (c *Converter) calculateFinalStats() error {
	// Get output file size
	outputStat, err := os.Stat(c.options.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to get output file size: %w", err)
	}
	c.stats.OutputFileSize = uint64(outputStat.Size())

	// Calculate compression ratio
	if c.stats.InputFileSize > 0 {
		c.stats.CompressionRatio = float64(c.stats.OutputFileSize) / float64(c.stats.InputFileSize)
	}

	// Calculate processing time
	c.stats.ProcessingTime = time.Since(c.startTime)

	return nil
}

// displayResults shows the conversion results
func (c *Converter) displayResults() {
	fmt.Printf("\nConversion completed successfully\n")
	fmt.Printf("================================================================\n")
	fmt.Printf("Conversion Summary\n")
	fmt.Printf("================================================================\n")

	// File sizes
	fmt.Printf("Input:         %s (%s)\n", filepath.Base(c.options.InputPath), humanize.Bytes(c.stats.InputFileSize))
	fmt.Printf("Output:        %s (%s)\n", filepath.Base(c.options.OutputPath), humanize.Bytes(c.stats.OutputFileSize))

	// Compression info
	if c.stats.CompressionRatio < 1.0 {
		fmt.Printf("Compression:   %.1f%% size reduction\n", (1.0-c.stats.CompressionRatio)*100)
	} else {
		fmt.Printf("Size change:   %.1f%% increase (likely due to text extraction)\n", (c.stats.CompressionRatio-1.0)*100)
	}

	// Content statistics
	fmt.Printf("Pages:         %d processed\n", c.stats.ProcessedPages)
	fmt.Printf("Text content:  %s characters\n", humanize.Comma(int64(c.stats.TextCharCount)))
	fmt.Printf("Target reader: %s\n", c.options.Profile.Name)

	// Performance
	fmt.Printf("Processing:    %v\n", c.stats.ProcessingTime.Round(time.Millisecond))

	fmt.Printf("================================================================\n")
	fmt.Printf("Ready for your %s\n", c.options.Profile.Name)
}


// GetStats returns the current conversion statistics
func (c *Converter) GetStats() ConversionStats {
	return c.stats
}

// cleanup closes resources
func (c *Converter) cleanup() {
	if c.pdfProc != nil {
		c.pdfProc.Close()
	}
}

