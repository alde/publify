package converter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alde/publify/pkg/reader"
	"github.com/bmaupin/go-epub"
)

// EPUBGenerator handles EPUB file creation
type EPUBGenerator struct {
	epub    *epub.Epub
	profile reader.Profile
	options EPUBOptions
}

// EPUBOptions defines EPUB generation settings
type EPUBOptions struct {
	Title       string
	Author      string
	Language    string
	Identifier  string
	Description string
	CoverPath   string
}

// NewEPUBGenerator creates a new EPUB generator
func NewEPUBGenerator(profile reader.Profile, opts EPUBOptions) *EPUBGenerator {
	e := epub.NewEpub(opts.Title)

	// Set metadata
	if opts.Author != "" {
		e.SetAuthor(opts.Author)
	}
	if opts.Language != "" {
		e.SetLang(opts.Language)
	} else {
		e.SetLang("en") // Default to English
	}
	if opts.Identifier != "" {
		e.SetIdentifier(opts.Identifier)
	}
	if opts.Description != "" {
		e.SetDescription(opts.Description)
	}

	// Add generator metadata
	e.SetPpd("publify-cli")
	e.SetDescription(opts.Description + " (Generated with Publify CLI)")

	return &EPUBGenerator{
		epub:    e,
		profile: profile,
		options: opts,
	}
}

// AddChapter adds a chapter to the EPUB from PDF pages
func (eg *EPUBGenerator) AddChapter(title string, pages []PDFPage) error {
	if len(pages) == 0 {
		return fmt.Errorf("no pages provided for chapter '%s'", title)
	}

	// Process text from all pages
	textProcessor := NewTextProcessor(TextProcessingOptions{
		PreserveFormatting: true,
		MinimizeFileSize:   true,
		ConvertToHTML:      true,
	})

	var allText strings.Builder
	for _, page := range pages {
		if page.HasText {
			processedText := textProcessor.ProcessText(page.Text)
			if processedText != "" {
				allText.WriteString(processedText)
				allText.WriteString("\n\n")
			}
		}
	}

	content := allText.String()
	if content == "" {
		content = "<p>No text content found on these pages.</p>"
	}

	// Create HTML content with proper structure
	htmlContent := eg.createHTMLContent(title, content)

	// Add chapter to EPUB
	_, err := eg.epub.AddSection(htmlContent, title, "", "")
	if err != nil {
		return fmt.Errorf("failed to add chapter '%s': %w", title, err)
	}

	return nil
}

// AddPage adds a single page as a chapter (legacy method, prefer AddChapter for better organization)
func (eg *EPUBGenerator) AddPage(page PDFPage) error {
	return eg.AddChapter("Chapter", []PDFPage{page})
}

func (eg *EPUBGenerator) createHTMLContent(title, content string) string {
	// Only add h1 title if it's not generic
	if title == "Chapter" {
		return content // Skip generic titles to avoid repetitive headings
	}

	html := fmt.Sprintf(`<h1>%s</h1>
%s`, title, content)

	return html
}

// SetCover sets the cover image for the EPUB
func (eg *EPUBGenerator) SetCover(imagePath string) error {
	if imagePath == "" {
		return nil
	}

	// Process image according to reader profile
	processedPath, err := eg.processImage(imagePath)
	if err != nil {
		return fmt.Errorf("failed to process cover image: %w", err)
	}

	// Add cover to EPUB
	_, err = eg.epub.AddImage(processedPath, "cover.jpg")
	if err != nil {
		return fmt.Errorf("failed to add cover image: %w", err)
	}

	return nil
}

// processImage optimizes an image for the target reader
func (eg *EPUBGenerator) processImage(imagePath string) (string, error) {
	tempDir, err := os.MkdirTemp("", "publify-images-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	processor := NewImageProcessor(eg.profile, tempDir)

	optimizedPath, err := processor.ProcessImage(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to process image: %w", err)
	}

	return optimizedPath, nil
}

func (eg *EPUBGenerator) AddMetadata(name, value string) {
	switch name {
	case "title":
		eg.epub.SetTitle(value)
	case "author":
		eg.epub.SetAuthor(value)
	case "description":
		eg.epub.SetDescription(value)
	case "language":
		eg.epub.SetLang(value)
	}
}

// Write saves the EPUB to the specified path
func (eg *EPUBGenerator) Write(outputPath string) error {
	dir := filepath.Dir(outputPath)
	if dir != "." {
	}

	// Write the EPUB file
	err := eg.epub.Write(outputPath)
	if err != nil {
		return fmt.Errorf("failed to write EPUB file: %w", err)
	}

	return nil
}

// EPUBMetadata contains EPUB metadata information
type EPUBMetadata struct {
	Title       string
	Author      string
	Language    string
	Identifier  string
	Description string
	Publisher   string
	Created     time.Time
	Modified    time.Time
}

// GetMetadata returns the current EPUB metadata
func (eg *EPUBGenerator) GetMetadata() EPUBMetadata {
	return EPUBMetadata{
		Title:       eg.epub.Title(),
		Author:      eg.epub.Author(),
		Language:    eg.epub.Lang(),
		Identifier:  eg.epub.Identifier(),
		Description: eg.epub.Description(),
		Created:     time.Now(), // Placeholder
		Modified:    time.Now(), // Placeholder
	}
}

func (eg *EPUBGenerator) Validate() error {
	if eg.epub.Title() == "" {
		return fmt.Errorf("EPUB title is required")
	}

	return nil
}
