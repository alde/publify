package converter

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alde/publify/pkg/reader"
)

func TestNew(t *testing.T) {
	opts := Options{
		InputPath:   "test.pdf",
		OutputPath:  "test.epub",
		WorkerCount: 4,
		Verbose:     false,
	}

	converter := New(opts)

	if converter == nil {
		t.Fatal("New() returned nil")
	}

	if converter.options.InputPath != opts.InputPath {
		t.Errorf("Expected InputPath %s, got %s", opts.InputPath, converter.options.InputPath)
	}

	if converter.options.WorkerCount != opts.WorkerCount {
		t.Errorf("Expected WorkerCount %d, got %d", opts.WorkerCount, converter.options.WorkerCount)
	}

	if converter.startTime.IsZero() {
		t.Error("Start time should be set")
	}
}

func TestCreateEPUBOptions(t *testing.T) {
	opts := Options{
		InputPath: "/path/to/test-book.pdf",
	}

	converter := New(opts)
	epubOpts := converter.createEPUBOptions()

	if epubOpts.Title != "test-book" {
		t.Errorf("Expected title 'test-book', got '%s'", epubOpts.Title)
	}

	if epubOpts.Author != "Unknown Author" {
		t.Errorf("Expected author 'Unknown Author', got '%s'", epubOpts.Author)
	}

	if epubOpts.Language != "en" {
		t.Errorf("Expected language 'en', got '%s'", epubOpts.Language)
	}

	if epubOpts.Identifier == "" {
		t.Error("Identifier should not be empty")
	}
}

func TestGetStats(t *testing.T) {
	converter := New(Options{})

	// Set some test stats
	converter.stats.PageCount = 10
	converter.stats.TextCharCount = 5000
	converter.stats.ProcessingTime = time.Second * 5

	stats := converter.GetStats()

	if stats.PageCount != 10 {
		t.Errorf("Expected PageCount 10, got %d", stats.PageCount)
	}

	if stats.TextCharCount != 5000 {
		t.Errorf("Expected TextCharCount 5000, got %d", stats.TextCharCount)
	}

	if stats.ProcessingTime != time.Second*5 {
		t.Errorf("Expected ProcessingTime 5s, got %v", stats.ProcessingTime)
	}
}

func TestGenerateEPUBWithEmptyPages(t *testing.T) {
	profile := reader.Profile{
		Name:         "Test Reader",
		Manufacturer: "Test",
		Capabilities: reader.DeviceCapabilities{
			DefaultFontSize: 12,
		},
	}

	opts := Options{
		Profile: profile,
	}

	converter := New(opts)
	converter.epubGen = NewEPUBGenerator(profile, EPUBOptions{
		Title: "Test Book",
	})

	err := converter.generateEPUB([]PDFPage{})
	if err == nil {
		t.Error("Expected error when generating EPUB with empty pages")
	}

	expectedMsg := "no pages to convert"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestGenerateEPUBWithValidPages(t *testing.T) {
	profile := reader.Profile{
		Name:         "Test Reader",
		Manufacturer: "Test",
		Capabilities: reader.DeviceCapabilities{
			DefaultFontSize: 12,
		},
	}

	opts := Options{
		Profile: profile,
	}

	converter := New(opts)
	converter.epubGen = NewEPUBGenerator(profile, EPUBOptions{
		Title: "Test Book",
	})

	pages := []PDFPage{
		{
			Number:  1,
			Text:    "This is the first page of text content.",
			HasText: true,
		},
		{
			Number:  2,
			Text:    "This is the second page with more content.",
			HasText: true,
		},
	}

	err := converter.generateEPUB(pages)
	if err != nil {
		t.Errorf("Unexpected error generating EPUB: %v", err)
	}

	// Check that stats were updated
	if converter.stats.TextCharCount == 0 {
		t.Error("TextCharCount should be updated after processing pages")
	}

	if converter.stats.ChapterCount != 2 {
		t.Errorf("Expected ChapterCount 2, got %d", converter.stats.ChapterCount)
	}
}

func TestCleanup(t *testing.T) {
	converter := New(Options{})

	// Test cleanup with nil pdfProc (should not panic)
	converter.cleanup()

	// Test would require more complex setup for actual PDF processor
	// This tests the basic case where cleanup is called safely
}

func TestConversionStatsCalculation(t *testing.T) {
	// Create a temporary test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.epub")

	// Create a small test file
	testContent := "test content for file size calculation"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	converter := New(Options{
		OutputPath: testFile,
	})

	// Set some input stats
	converter.stats.InputFileSize = 1000

	err = converter.calculateFinalStats()
	if err != nil {
		t.Errorf("Unexpected error calculating final stats: %v", err)
	}

	if converter.stats.OutputFileSize == 0 {
		t.Error("OutputFileSize should be calculated")
	}

	if converter.stats.CompressionRatio == 0 {
		t.Error("CompressionRatio should be calculated")
	}

	if converter.stats.ProcessingTime == 0 {
		t.Error("ProcessingTime should be calculated")
	}
}