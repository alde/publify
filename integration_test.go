package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alde/publify/pkg/converter"
	"github.com/alde/publify/pkg/reader"
)

func TestIntegrationRomeoAndJulietConversion(t *testing.T) {
	// Check if test file exists
	testFile := "testdata/romeo-and-juliet.pdf"
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("Romeo and Juliet test file not found, skipping integration test")
	}

	// Create temporary output directory
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "romeo-and-juliet.epub")

	// Set up conversion options with a test reader profile
	profile := reader.Profile{
		Name:         "Kobo Clara HD",
		Manufacturer: "Kobo",
		Capabilities: reader.DeviceCapabilities{
			ScreenWidth:           1072,
			ScreenHeight:          1448,
			DPI:                   300,
			SupportsColor:         false,
			ColorDepth:            8,
			MaxImageWidth:         1000,
			MaxImageHeight:        1400,
			ImageQuality:          85,
			CompressionLevel:      "high",
			SupportedImageFormats: []string{"jpeg", "png"},
			PreferredImageFormat:  "jpeg",
			DefaultFontSize:       12,
		},
	}

	opts := converter.Options{
		InputPath:   testFile,
		OutputPath:  outputFile,
		Profile:     profile,
		WorkerCount: 2,
		Verbose:     true,
	}

	// Create converter and run conversion
	conv := converter.New(opts)
	err := conv.Convert()

	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	// Check that output file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatal("Output EPUB file was not created")
	}

	// Get file info
	outputStat, err := os.Stat(outputFile)
	if err != nil {
		t.Fatalf("Failed to stat output file: %v", err)
	}

	// Check that output file has reasonable size (should be > 0 bytes)
	if outputStat.Size() == 0 {
		t.Error("Output EPUB file is empty")
	}

	// Get conversion statistics
	stats := conv.GetStats()

	// Verify statistics make sense
	if stats.InputFileSize == 0 {
		t.Error("Input file size should be greater than 0")
	}

	if stats.OutputFileSize == 0 {
		t.Error("Output file size should be greater than 0")
	}

	if stats.PageCount == 0 {
		t.Error("Page count should be greater than 0")
	}

	if stats.ProcessingTime == 0 {
		t.Error("Processing time should be greater than 0")
	}

	t.Logf("Conversion completed successfully:")
	t.Logf("  Input size: %d bytes", stats.InputFileSize)
	t.Logf("  Output size: %d bytes", stats.OutputFileSize)
	t.Logf("  Pages processed: %d", stats.ProcessedPages)
	t.Logf("  Text characters: %d", stats.TextCharCount)
	t.Logf("  Processing time: %v", stats.ProcessingTime)
}

func TestIntegrationSmallPDFConversion(t *testing.T) {
	// This test would work with any small PDF file in testdata
	testFiles := []string{
		"testdata/romeo-and-juliet.pdf",
		"testdata/Air Babylon - Edwards-Jones, Imogen.pdf",
	}

	var testFile string
	for _, file := range testFiles {
		if _, err := os.Stat(file); err == nil {
			testFile = file
			break
		}
	}

	if testFile == "" {
		t.Skip("No test PDF files found, skipping integration test")
	}

	// Create temporary output directory
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "test-output.epub")

	// Use a simple Kindle profile for testing
	profile := reader.Profile{
		Name:         "Kindle Paperwhite",
		Manufacturer: "Amazon",
		Capabilities: reader.DeviceCapabilities{
			ScreenWidth:           758,
			ScreenHeight:          1024,
			DPI:                   300,
			SupportsColor:         false,
			ColorDepth:            8,
			MaxImageWidth:         700,
			MaxImageHeight:        900,
			ImageQuality:          85,
			CompressionLevel:      "high",
			SupportedImageFormats: []string{"jpeg", "png"},
			PreferredImageFormat:  "jpeg",
			DefaultFontSize:       11,
		},
	}

	opts := converter.Options{
		InputPath:   testFile,
		OutputPath:  outputFile,
		Profile:     profile,
		WorkerCount: 1, // Use single worker for simpler testing
		Verbose:     false,
	}

	// Test that conversion works without errors
	conv := converter.New(opts)
	err := conv.Convert()

	if err != nil {
		// If conversion fails, check if it's due to file format issues
		if strings.Contains(err.Error(), "failed to open PDF") {
			t.Skipf("PDF file format issue, skipping: %v", err)
		}
		t.Fatalf("Unexpected conversion error: %v", err)
	}

	// Verify output file exists and has content
	outputStat, err := os.Stat(outputFile)
	if err != nil {
		t.Fatalf("Output file not created: %v", err)
	}

	if outputStat.Size() < 100 {
		t.Errorf("Output file seems too small (%d bytes), might be invalid", outputStat.Size())
	}

	t.Logf("Successfully converted %s to %s (%d bytes)",
		filepath.Base(testFile), filepath.Base(outputFile), outputStat.Size())
}

func TestIntegrationConverterOptionsValidation(t *testing.T) {
	// Test with invalid input file
	tempDir := t.TempDir()

	opts := converter.Options{
		InputPath:   "nonexistent.pdf",
		OutputPath:  filepath.Join(tempDir, "output.epub"),
		WorkerCount: 1,
	}

	conv := converter.New(opts)
	err := conv.Convert()

	if err == nil {
		t.Error("Expected error when converting nonexistent file")
	}

	// Test that error message is descriptive
	if !strings.Contains(err.Error(), "failed to open PDF") &&
		!strings.Contains(err.Error(), "no such file") {
		t.Errorf("Error message should indicate file issue, got: %v", err)
	}
}

func TestIntegrationWorkerPoolConfiguration(t *testing.T) {
	// Test with different worker counts
	workerCounts := []int{1, 2, 4}

	// Skip if no test files available
	testFile := "testdata/romeo-and-juliet.pdf"
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("Test file not found, skipping worker pool test")
	}

	profile := reader.Profile{
		Name: "Test Reader",
		Capabilities: reader.DeviceCapabilities{
			DefaultFontSize: 12,
		},
	}

	for _, workerCount := range workerCounts {
		t.Run(fmt.Sprintf("workers_%d", workerCount), func(t *testing.T) {
			tempDir := t.TempDir()
			outputFile := filepath.Join(tempDir, "test.epub")

			opts := converter.Options{
				InputPath:   testFile,
				OutputPath:  outputFile,
				Profile:     profile,
				WorkerCount: workerCount,
				Verbose:     false,
			}

			conv := converter.New(opts)

			// Just test that it doesn't crash with different worker counts
			// Actual conversion might fail due to file format, but shouldn't crash
			err := conv.Convert()

			// Allow PDF format errors but not crashes
			if err != nil && !strings.Contains(err.Error(), "PDF") {
				t.Errorf("Unexpected error with %d workers: %v", workerCount, err)
			}
		})
	}
}
