package converter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alde/publify/pkg/reader"
)

func TestNewEPUBGenerator(t *testing.T) {
	profile := reader.Profile{
		Name:         "Test Reader",
		Manufacturer: "Test Company",
		Capabilities: reader.DeviceCapabilities{
			DefaultFontSize: 14,
		},
	}

	opts := EPUBOptions{
		Title:       "Test Book",
		Author:      "Test Author",
		Language:    "en",
		Identifier:  "test-id-123",
		Description: "A test book for testing",
	}

	generator := NewEPUBGenerator(profile, opts)

	if generator == nil {
		t.Fatal("NewEPUBGenerator returned nil")
	}

	if generator.profile.Name != profile.Name {
		t.Errorf("Expected profile name %s, got %s", profile.Name, generator.profile.Name)
	}

	if generator.options.Title != opts.Title {
		t.Errorf("Expected title %s, got %s", opts.Title, generator.options.Title)
	}
}

func TestEPUBGeneratorValidate(t *testing.T) {
	profile := reader.Profile{
		Name: "Test Reader",
		Capabilities: reader.DeviceCapabilities{
			DefaultFontSize: 12,
		},
	}

	// Test with valid options
	opts := EPUBOptions{
		Title: "Valid Book",
	}
	generator := NewEPUBGenerator(profile, opts)

	err := generator.Validate()
	if err != nil {
		t.Errorf("Unexpected validation error: %v", err)
	}

	// Test with empty title
	emptyOpts := EPUBOptions{
		Title: "",
	}
	emptyGenerator := NewEPUBGenerator(profile, emptyOpts)

	err = emptyGenerator.Validate()
	if err == nil {
		t.Error("Expected validation error for empty title")
	}
}

func TestEPUBGeneratorAddPage(t *testing.T) {
	profile := reader.Profile{
		Name: "Test Reader",
		Capabilities: reader.DeviceCapabilities{
			DefaultFontSize: 12,
		},
	}

	opts := EPUBOptions{
		Title: "Test Book",
	}

	generator := NewEPUBGenerator(profile, opts)

	page := PDFPage{
		Number:  1,
		Text:    "This is some test content for the first page.",
		HasText: true,
	}

	err := generator.AddPage(page)
	if err != nil {
		t.Errorf("Unexpected error adding page: %v", err)
	}
}

func TestEPUBGeneratorAddChapter(t *testing.T) {
	profile := reader.Profile{
		Name: "Test Reader",
		Capabilities: reader.DeviceCapabilities{
			DefaultFontSize: 12,
		},
	}

	opts := EPUBOptions{
		Title: "Test Book",
	}

	generator := NewEPUBGenerator(profile, opts)

	// Test with valid pages
	pages := []PDFPage{
		{
			Number:  1,
			Text:    "First page content.",
			HasText: true,
		},
		{
			Number:  2,
			Text:    "Second page content.",
			HasText: true,
		},
	}

	err := generator.AddChapter("Chapter 1", pages)
	if err != nil {
		t.Errorf("Unexpected error adding chapter: %v", err)
	}

	// Test with empty pages
	err = generator.AddChapter("Empty Chapter", []PDFPage{})
	if err == nil {
		t.Error("Expected error when adding chapter with no pages")
	}
}

func TestEPUBGeneratorWrite(t *testing.T) {
	profile := reader.Profile{
		Name: "Test Reader",
		Capabilities: reader.DeviceCapabilities{
			DefaultFontSize: 12,
		},
	}

	opts := EPUBOptions{
		Title: "Test Book",
	}

	generator := NewEPUBGenerator(profile, opts)

	// Add some content
	page := PDFPage{
		Number:  1,
		Text:    "Test content",
		HasText: true,
	}
	generator.AddPage(page)

	// Write to temporary file
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "test.epub")

	err := generator.Write(outputPath)
	if err != nil {
		t.Errorf("Unexpected error writing EPUB: %v", err)
	}

	// Check that file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("EPUB file was not created")
	}
}

func TestEPUBGeneratorGetMetadata(t *testing.T) {
	profile := reader.Profile{
		Name: "Test Reader",
		Capabilities: reader.DeviceCapabilities{
			DefaultFontSize: 12,
		},
	}

	opts := EPUBOptions{
		Title:       "Test Book",
		Author:      "Test Author",
		Language:    "en",
		Identifier:  "test-123",
		Description: "Test description",
	}

	generator := NewEPUBGenerator(profile, opts)
	metadata := generator.GetMetadata()

	if metadata.Title != opts.Title {
		t.Errorf("Expected title %s, got %s", opts.Title, metadata.Title)
	}

	if metadata.Author != opts.Author {
		t.Errorf("Expected author %s, got %s", opts.Author, metadata.Author)
	}

	if metadata.Language != opts.Language {
		t.Errorf("Expected language %s, got %s", opts.Language, metadata.Language)
	}
}

func TestCreateHTMLContent(t *testing.T) {
	profile := reader.Profile{
		Name: "Test Reader",
		Capabilities: reader.DeviceCapabilities{
			DefaultFontSize: 14,
		},
	}

	opts := EPUBOptions{
		Title: "Test Book",
	}

	generator := NewEPUBGenerator(profile, opts)

	title := "Test Chapter"
	content := "<p>This is test content.</p>"

	html := generator.createHTMLContent(title, content)

	// Check that HTML contains expected elements
	if !containsString(html, title) {
		t.Error("HTML should contain the chapter title")
	}

	if !containsString(html, content) {
		t.Error("HTML should contain the provided content")
	}

	// Check for basic structure (should be simple body content only)
	if !containsString(html, "<h1>") {
		t.Error("HTML should have h1 tag for title")
	}

	// Should NOT contain full HTML document structure (go-epub handles that)
	if containsString(html, "<!DOCTYPE html") {
		t.Error("HTML should not have DOCTYPE declaration (go-epub adds this)")
	}

	if containsString(html, "<html xmlns=") {
		t.Error("HTML should not have html tag (go-epub adds this)")
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) &&
		   (s == substr ||
		    len(s) > len(substr) &&
		    (s[:len(substr)] == substr ||
		     s[len(s)-len(substr):] == substr ||
		     containsString(s[1:], substr)))
}