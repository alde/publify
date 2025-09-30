package converter

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/alde/publify/internal/worker"
	"github.com/alde/publify/pkg/reader"
	"github.com/dustin/go-humanize"
)

// Options contains conversion settings (because even PDFs need their preferences, ja?)
type Options struct {
	InputPath      string
	OutputPath     string
	Profile        reader.Profile
	WorkerCount    int
	Verbose        bool
	EnableOCR      bool
	OCRLanguage    string
	ImagePageRange string
	SkipPages      string
}

// Converter handles the PDF to EPUB conversion process (with the thoroughness of a Swedish quality inspector)
type Converter struct {
	options   Options
	pdfProc   *PDFProcessor
	epubGen   *EPUBGenerator
	stats     ConversionStats
	startTime time.Time
}

// ConversionStats tracks conversion metrics (numbers that make developers feel accomplished)
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

	// Get input file size for statistics (because size matters in file conversion, unlike in many other things)
	inputSize, err := c.pdfProc.GetFileSize()
	if err != nil {
		return fmt.Errorf("failed to get input file size: %w", err)
	}
	c.stats.InputFileSize = uint64(inputSize)

	// Create worker pool with progress tracking (Swedish efficiency meets Go concurrency)
	pool := worker.NewPoolWithProgress(c.options.WorkerCount, c.pdfProc.GetPageCount())
	pool.Start()
	defer pool.Stop()

	if c.options.Verbose {
		fmt.Printf("Starting conversion of %s to %s\n", c.options.InputPath, c.options.OutputPath)
		fmt.Printf("Target reader: %s (%s)\n", c.options.Profile.Name, c.options.Profile.Manufacturer)
		fmt.Printf("Using %d worker goroutines\n", pool.WorkerCount())
	}

	// Process PDF pages (where the magic happens, or at least where we pretend it does)
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
	pdfProc, err := NewPDFProcessor(c.options.InputPath, c.options.ImagePageRange, c.options.EnableOCR, c.options.OCRLanguage, c.options.SkipPages)
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

// generateEPUB creates the EPUB content from processed pages
func (c *Converter) generateEPUB(pages []PDFPage) error {
	if len(pages) == 0 {
		return fmt.Errorf("no pages to convert")
	}

	// Group pages into reasonable chapters (because nobody wants 200 tiny chapters)
	chapters := c.groupPagesIntoChapters(pages)

	for i, chapter := range chapters {
		chapterTitle := fmt.Sprintf("Chapter %d", i+1)
		if err := c.epubGen.AddChapter(chapterTitle, chapter); err != nil {
			return fmt.Errorf("failed to add chapter %d: %w", i+1, err)
		}

		// Update statistics
		for _, page := range chapter {
			c.stats.TextCharCount += len(page.Text)
		}
		c.stats.ChapterCount++
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

	// Validation results
	if c.pdfProc != nil {
		rejectedPages := c.pdfProc.GetRejectedPages()
		if len(rejectedPages) > 0 {
			fmt.Printf("\n")
			fmt.Printf("Validation Results:\n")
			fmt.Printf("Pages rejected by bleed-through detection: %v\n", rejectedPages)
			fmt.Printf("Suggestion: Consider adding --skip \"%s\" for faster processing\n", formatPageList(rejectedPages))
		}
	}

	fmt.Printf("================================================================\n")
	fmt.Printf("Ready for your %s\n", c.options.Profile.Name)
}

// formatPageList formats a list of page numbers into a comma-separated string
func formatPageList(pages []int) string {
	if len(pages) == 0 {
		return ""
	}

	strs := make([]string, len(pages))
	for i, page := range pages {
		strs[i] = fmt.Sprintf("%d", page)
	}
	return strings.Join(strs, ",")
}

// GetStats returns the current conversion statistics
func (c *Converter) GetStats() ConversionStats {
	return c.stats
}

// groupPagesIntoChapters intelligently groups pages into chapters for better reading experience
func (c *Converter) groupPagesIntoChapters(pages []PDFPage) [][]PDFPage {
	const maxPagesPerChapter = 15 // Reasonable chapter size (increased for books with many short pages)
	const minTextPerChapter = 800 // Minimum characters for a meaningful chapter

	var chapters [][]PDFPage
	var currentChapter []PDFPage
	currentTextLength := 0

	for i, page := range pages {
		// Check if this page starts with a potential chapter marker
		isChapterBreak := false
		if page.HasText && len(currentChapter) > 0 {
			// Look for time spans or traditional chapter markers at the start of the page
			lines := strings.Split(strings.TrimSpace(page.Text), "\n")
			if len(lines) > 0 {
				firstLine := strings.TrimSpace(lines[0])
				// Check for time spans (like "5-6am") or traditional chapter markers
				if c.isTimeSpanChapterMarker(firstLine) ||
					strings.Contains(strings.ToLower(firstLine), "chapter") {
					isChapterBreak = true
				}
			}
		}

		// Add page to current chapter
		currentChapter = append(currentChapter, page)
		if page.HasText {
			currentTextLength += len(page.Text)
		}

		// Create new chapter if we've reached limits or found a natural break
		shouldBreak := isChapterBreak ||
			len(currentChapter) >= maxPagesPerChapter ||
			(currentTextLength >= minTextPerChapter && len(currentChapter) >= 3)

		// Don't break on the first page or if we'd create a tiny chapter
		if shouldBreak && i > 0 && len(currentChapter) > 1 {
			chapters = append(chapters, currentChapter)
			currentChapter = []PDFPage{}
			currentTextLength = 0
		}
	}

	// Add remaining pages as final chapter
	if len(currentChapter) > 0 {
		chapters = append(chapters, currentChapter)
	}

	// Ensure we have at least one chapter
	if len(chapters) == 0 {
		chapters = [][]PDFPage{pages}
	}

	return chapters
}

// isTimeSpanChapterMarker detects time-based chapter markers like "5-6am"
func (c *Converter) isTimeSpanChapterMarker(line string) bool {
	line = strings.ToLower(strings.TrimSpace(line))

	// Patterns like "5-6am", "11pm-12am", "2.30-3.30pm"
	timePatterns := []string{
		`^\d{1,2}(:\d{2})?(-|\s*to\s*)\d{1,2}(:\d{2})?(am|pm)$`,
		`^\d{1,2}(:\d{2})?(am|pm)(-|\s*to\s*)\d{1,2}(:\d{2})?(am|pm)$`,
		`^\d{1,2}\.\d{2}(-|\s*to\s*)\d{1,2}\.\d{2}(am|pm)$`,
	}

	for _, pattern := range timePatterns {
		if regexp.MustCompile(pattern).MatchString(line) {
			return true
		}
	}

	return false
}

// cleanup closes resources
func (c *Converter) cleanup() {
	if c.pdfProc != nil {
		c.pdfProc.Close()
	}
}
