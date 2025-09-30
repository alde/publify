package metadata

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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
	CoverPath   string
}

// EPUBReader provides read-only access to EPUB metadata
type EPUBReader struct {
	filePath  string
	zipReader *zip.ReadCloser
}

// EPUBEditor provides read-write access to EPUB metadata
type EPUBEditor struct {
	filePath string
	tempDir  string
	metadata EPUBMetadata
	modified bool
}

// Chapter represents a chapter in the EPUB
type Chapter struct {
	ID    string
	Title string
	Path  string
}

// NewEPUBReader creates a new EPUB reader
func NewEPUBReader(filePath string) (*EPUBReader, error) {
	zipReader, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open EPUB file: %w", err)
	}

	return &EPUBReader{
		filePath:  filePath,
		zipReader: zipReader,
	}, nil
}

// Close closes the EPUB reader
func (r *EPUBReader) Close() error {
	if r.zipReader != nil {
		return r.zipReader.Close()
	}
	return nil
}

// GetMetadata reads metadata from the EPUB
func (r *EPUBReader) GetMetadata() (EPUBMetadata, error) {
	// Find and read the OPF file
	opfPath, err := r.findOPFFile()
	if err != nil {
		return EPUBMetadata{}, fmt.Errorf("failed to find OPF file: %w", err)
	}

	opfContent, err := r.readFileFromZip(opfPath)
	if err != nil {
		return EPUBMetadata{}, fmt.Errorf("failed to read OPF file: %w", err)
	}

	// Parse metadata from OPF
	metadata, err := parseOPFMetadata(opfContent)
	if err != nil {
		return EPUBMetadata{}, fmt.Errorf("failed to parse metadata: %w", err)
	}

	// Get file timestamps
	stat, err := os.Stat(r.filePath)
	if err == nil {
		metadata.Modified = stat.ModTime()
	}

	return metadata, nil
}

// GetChapterList returns a list of chapters in the EPUB
func (r *EPUBReader) GetChapterList() ([]Chapter, error) {
	// Find and read the OPF file
	opfPath, err := r.findOPFFile()
	if err != nil {
		return nil, fmt.Errorf("failed to find OPF file: %w", err)
	}

	opfContent, err := r.readFileFromZip(opfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read OPF file: %w", err)
	}

	// Parse chapter list from OPF
	chapters, err := parseOPFChapters(opfContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse chapters: %w", err)
	}

	return chapters, nil
}

// findOPFFile locates the OPF file within the EPUB
func (r *EPUBReader) findOPFFile() (string, error) {
	// First, check META-INF/container.xml
	containerContent, err := r.readFileFromZip("META-INF/container.xml")
	if err != nil {
		return "", fmt.Errorf("failed to read container.xml: %w", err)
	}

	// Parse container.xml to find OPF path
	type Container struct {
		Rootfiles struct {
			Rootfile []struct {
				FullPath string `xml:"full-path,attr"`
			} `xml:"rootfile"`
		} `xml:"rootfiles"`
	}

	var container Container
	if err := xml.Unmarshal(containerContent, &container); err != nil {
		return "", fmt.Errorf("failed to parse container.xml: %w", err)
	}

	if len(container.Rootfiles.Rootfile) == 0 {
		return "", fmt.Errorf("no rootfile found in container.xml")
	}

	return container.Rootfiles.Rootfile[0].FullPath, nil
}

// readFileFromZip reads a file from within the ZIP archive
func (r *EPUBReader) readFileFromZip(path string) ([]byte, error) {
	for _, file := range r.zipReader.File {
		if file.Name == path {
			rc, err := file.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("file not found: %s", path)
}

// parseOPFMetadata extracts metadata from OPF content
func parseOPFMetadata(opfContent []byte) (EPUBMetadata, error) {
	// Simple OPF structure for metadata parsing
	type OPF struct {
		Metadata struct {
			Title       []string `xml:"title"`
			Creator     []string `xml:"creator"`
			Language    []string `xml:"language"`
			Identifier  []string `xml:"identifier"`
			Description []string `xml:"description"`
			Publisher   []string `xml:"publisher"`
			Date        []string `xml:"date"`
		} `xml:"metadata"`
	}

	var opf OPF
	if err := xml.Unmarshal(opfContent, &opf); err != nil {
		return EPUBMetadata{}, fmt.Errorf("failed to parse OPF XML: %w", err)
	}

	metadata := EPUBMetadata{}

	if len(opf.Metadata.Title) > 0 {
		metadata.Title = opf.Metadata.Title[0]
	}
	if len(opf.Metadata.Creator) > 0 {
		metadata.Author = opf.Metadata.Creator[0]
	}
	if len(opf.Metadata.Language) > 0 {
		metadata.Language = opf.Metadata.Language[0]
	}
	if len(opf.Metadata.Identifier) > 0 {
		metadata.Identifier = opf.Metadata.Identifier[0]
	}
	if len(opf.Metadata.Description) > 0 {
		metadata.Description = opf.Metadata.Description[0]
	}
	if len(opf.Metadata.Publisher) > 0 {
		metadata.Publisher = opf.Metadata.Publisher[0]
	}

	// Parse date if available
	if len(opf.Metadata.Date) > 0 {
		if created, err := time.Parse(time.RFC3339, opf.Metadata.Date[0]); err == nil {
			metadata.Created = created
		}
	}

	return metadata, nil
}

// parseOPFChapters extracts chapter information from OPF content
func parseOPFChapters(opfContent []byte) ([]Chapter, error) {
	// Simple parsing - in a full implementation this would be more robust
	type OPF struct {
		Spine struct {
			ItemRef []struct {
				IDRef string `xml:"idref,attr"`
			} `xml:"itemref"`
		} `xml:"spine"`
		Manifest struct {
			Item []struct {
				ID   string `xml:"id,attr"`
				Href string `xml:"href,attr"`
			} `xml:"item"`
		} `xml:"manifest"`
	}

	var opf OPF
	if err := xml.Unmarshal(opfContent, &opf); err != nil {
		return nil, fmt.Errorf("failed to parse OPF XML: %w", err)
	}

	// Create a map of ID to href for quick lookup
	idToHref := make(map[string]string)
	for _, item := range opf.Manifest.Item {
		idToHref[item.ID] = item.Href
	}

	// Build chapter list from spine
	var chapters []Chapter
	for i, itemRef := range opf.Spine.ItemRef {
		if href, exists := idToHref[itemRef.IDRef]; exists {
			chapter := Chapter{
				ID:    itemRef.IDRef,
				Title: fmt.Sprintf("Chapter %d", i+1), // Simple title
				Path:  href,
			}
			chapters = append(chapters, chapter)
		}
	}

	return chapters, nil
}

// NewEPUBEditor creates a new EPUB editor
func NewEPUBEditor(filePath string) (*EPUBEditor, error) {
	// Read current metadata
	reader, err := NewEPUBReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open EPUB for reading: %w", err)
	}
	defer reader.Close()

	metadata, err := reader.GetMetadata()
	if err != nil {
		return nil, fmt.Errorf("failed to read current metadata: %w", err)
	}

	// Create temporary directory for editing
	tempDir, err := os.MkdirTemp("", "publify-epub-edit-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	return &EPUBEditor{
		filePath: filePath,
		tempDir:  tempDir,
		metadata: metadata,
		modified: false,
	}, nil
}

// Close cleans up the EPUB editor
func (e *EPUBEditor) Close() error {
	if e.tempDir != "" {
		return os.RemoveAll(e.tempDir)
	}
	return nil
}

// SetTitle sets the book title
func (e *EPUBEditor) SetTitle(title string) error {
	e.metadata.Title = title
	e.modified = true
	return nil
}

// SetAuthor sets the book author
func (e *EPUBEditor) SetAuthor(author string) error {
	e.metadata.Author = author
	e.modified = true
	return nil
}

// SetDescription sets the book description
func (e *EPUBEditor) SetDescription(description string) error {
	e.metadata.Description = description
	e.modified = true
	return nil
}

// SetLanguage sets the book language
func (e *EPUBEditor) SetLanguage(language string) error {
	e.metadata.Language = language
	e.modified = true
	return nil
}

// SetPublisher sets the book publisher
func (e *EPUBEditor) SetPublisher(publisher string) error {
	e.metadata.Publisher = publisher
	e.modified = true
	return nil
}

// SetCover sets the book cover image
func (e *EPUBEditor) SetCover(coverPath string) error {
	// Copy cover image to temp directory
	coverExt := strings.ToLower(filepath.Ext(coverPath))
	tempCoverPath := filepath.Join(e.tempDir, "cover"+coverExt)

	if err := copyFile(coverPath, tempCoverPath); err != nil {
		return fmt.Errorf("failed to copy cover image: %w", err)
	}

	e.metadata.CoverPath = tempCoverPath
	e.modified = true
	return nil
}

// Save saves the changes to the EPUB file
func (e *EPUBEditor) Save() error {
	if !e.modified {
		return nil // No changes to save
	}

	// For this implementation, we'll create a simple message indicating
	// that the metadata would be saved. A full implementation would:
	// 1. Extract the EPUB to temp directory
	// 2. Modify the OPF file with new metadata
	// 3. Add cover image if provided
	// 4. Repackage as EPUB
	// 5. Replace original file

	e.metadata.Modified = time.Now()

	// Placeholder for actual EPUB modification
	fmt.Printf("Note: Full EPUB metadata editing not yet implemented in this version.\n")
	fmt.Printf("Metadata that would be saved:\n")
	fmt.Printf("  Title: %s\n", e.metadata.Title)
	fmt.Printf("  Author: %s\n", e.metadata.Author)
	if e.metadata.Description != "" {
		fmt.Printf("  Description: %s\n", e.metadata.Description)
	}
	if e.metadata.Language != "" {
		fmt.Printf("  Language: %s\n", e.metadata.Language)
	}
	if e.metadata.Publisher != "" {
		fmt.Printf("  Publisher: %s\n", e.metadata.Publisher)
	}
	if e.metadata.CoverPath != "" {
		fmt.Printf("  Cover: %s\n", e.metadata.CoverPath)
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(sourceFile)
	return err
}
