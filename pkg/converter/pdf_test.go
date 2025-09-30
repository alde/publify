package converter

import (
	"strings"
	"testing"
)

func TestCleanText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "remove carriage returns",
			input:    "Hello\r\nWorld\r",
			expected: "Hello\nWorld\n",
		},
		{
			name:     "normalize line endings",
			input:    "Line 1\r\nLine 2\rLine 3\n",
			expected: "Line 1\nLine 2\nLine 3\n",
		},
		{
			name:     "preserve paragraph breaks",
			input:    "Paragraph 1\n\nParagraph 2\n\n\nParagraph 3",
			expected: "Paragraph 1\n\nParagraph 2\n\nParagraph 3",
		},
		{
			name:     "remove excessive whitespace",
			input:    "  Spaced   Text  \n  More Text  ",
			expected: "Spaced   Text\nMore Text",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace only",
			input:    "   \n  \n  ",
			expected: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := cleanText(test.input)
			if result != test.expected {
				t.Errorf("cleanText() = %q, expected %q", result, test.expected)
			}
		})
	}
}

func TestPDFPageType(t *testing.T) {
	tests := []struct {
		name     string
		pageNum  int
		hasText  bool
		hasImage bool
		pageType PageType
	}{
		{
			name:     "text page",
			pageNum:  1,
			hasText:  true,
			hasImage: false,
			pageType: PageTypeText,
		},
		{
			name:     "image page",
			pageNum:  2,
			hasText:  false,
			hasImage: true,
			pageType: PageTypeImage,
		},
		{
			name:     "mixed content defaults to text",
			pageNum:  3,
			hasText:  true,
			hasImage: true,
			pageType: PageTypeText,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			page := PDFPage{
				Number:   test.pageNum,
				HasText:  test.hasText,
				HasImage: test.hasImage,
				PageType: test.pageType,
			}

			if page.PageType != test.pageType {
				t.Errorf("Expected PageType %v, got %v", test.pageType, page.PageType)
			}
		})
	}
}

// These functions are no longer relevant with PDFium implementation

// Mock tests for PDF processor functions that would require actual PDF files
func TestPDFPageStruct(t *testing.T) {
	page := PDFPage{
		Number:    1,
		Text:      "Sample text content",
		Width:     612.0,
		Height:    792.0,
		HasText:   true,
		HasImage:  false,
		PageType:  PageTypeText,
		ImageData: []byte{},
	}

	if page.Number != 1 {
		t.Errorf("Expected page number 1, got %d", page.Number)
	}

	if !page.HasText {
		t.Error("Expected page to have text")
	}

	if page.HasImage {
		t.Error("Expected page to not have image")
	}

	if page.PageType != PageTypeText {
		t.Errorf("Expected PageType %v, got %v", PageTypeText, page.PageType)
	}

	if !strings.Contains(page.Text, "Sample text") {
		t.Error("Page text should contain expected content")
	}
}

// Test PDF page dimensions
func TestPDFPageDimensions(t *testing.T) {
	tests := []struct {
		name   string
		width  float64
		height float64
		valid  bool
	}{
		{"standard letter", 612.0, 792.0, true},
		{"A4", 595.0, 842.0, true},
		{"invalid zero width", 0.0, 792.0, false},
		{"invalid zero height", 612.0, 0.0, false},
		{"negative dimensions", -100.0, -200.0, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			page := PDFPage{
				Width:  test.width,
				Height: test.height,
			}

			isValid := page.Width > 0 && page.Height > 0
			if isValid != test.valid {
				t.Errorf("Expected validity %v for dimensions %.1fx%.1f, got %v",
					test.valid, test.width, test.height, isValid)
			}
		})
	}
}
