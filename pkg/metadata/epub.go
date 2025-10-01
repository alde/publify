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
	newCover string // Track if a new cover was explicitly set
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
			Meta        []struct {
				Name    string `xml:"name,attr"`
				Content string `xml:"content,attr"`
			} `xml:"meta"`
		} `xml:"metadata"`
		Manifest struct {
			Item []struct {
				ID         string `xml:"id,attr"`
				Href       string `xml:"href,attr"`
				Properties string `xml:"properties,attr"`
			} `xml:"item"`
		} `xml:"manifest"`
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

	// Extract cover path from metadata and manifest
	var coverItemID string

	// First, look for cover metadata
	for _, meta := range opf.Metadata.Meta {
		if meta.Name == "cover" {
			coverItemID = meta.Content
			break
		}
	}

	// If we found a cover item ID, look up its path in the manifest
	if coverItemID != "" {
		for _, item := range opf.Manifest.Item {
			if item.ID == coverItemID {
				metadata.CoverPath = item.Href
				break
			}
		}
	}

	// Alternative: look for items with cover-image property
	if metadata.CoverPath == "" {
		for _, item := range opf.Manifest.Item {
			if strings.Contains(item.Properties, "cover-image") {
				metadata.CoverPath = item.Href
				break
			}
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

	e.newCover = tempCoverPath
	e.modified = true
	return nil
}

// Save saves the changes to the EPUB file
func (e *EPUBEditor) Save() error {
	if !e.modified {
		return nil // No changes to save
	}

	// 1. Extract EPUB to temp directory
	extractDir := filepath.Join(e.tempDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return fmt.Errorf("failed to create extraction directory: %w", err)
	}

	if err := e.extractEPUB(extractDir); err != nil {
		return fmt.Errorf("failed to extract EPUB: %w", err)
	}

	// 2. Find and modify the OPF file
	if err := e.updateOPFMetadata(extractDir); err != nil {
		return fmt.Errorf("failed to update OPF metadata: %w", err)
	}

	// 3. Add cover image if provided
	if e.newCover != "" {
		if err := e.updateCoverImage(extractDir); err != nil {
			return fmt.Errorf("failed to update cover image: %w", err)
		}
	}

	// 4. Repackage as EPUB
	newEPUBPath := e.filePath + ".new"
	if err := e.repackageEPUB(extractDir, newEPUBPath); err != nil {
		return fmt.Errorf("failed to repackage EPUB: %w", err)
	}

	// 5. Replace original file
	if err := os.Rename(newEPUBPath, e.filePath); err != nil {
		return fmt.Errorf("failed to replace original file: %w", err)
	}

	e.metadata.Modified = time.Now()
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

// extractEPUB extracts the EPUB file to the specified directory
func (e *EPUBEditor) extractEPUB(extractDir string) error {
	zipReader, err := zip.OpenReader(e.filePath)
	if err != nil {
		return fmt.Errorf("failed to open EPUB: %w", err)
	}
	defer zipReader.Close()

	for _, file := range zipReader.File {
		filePath := filepath.Join(extractDir, file.Name)

		// Create directory if needed
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(filePath, file.FileInfo().Mode()); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", filePath, err)
			}
			continue
		}

		// Create parent directories
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory for %s: %w", filePath, err)
		}

		// Extract file
		src, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open file %s in ZIP: %w", file.Name, err)
		}

		dst, err := os.Create(filePath)
		if err != nil {
			src.Close()
			return fmt.Errorf("failed to create file %s: %w", filePath, err)
		}

		_, err = io.Copy(dst, src)
		src.Close()
		dst.Close()

		if err != nil {
			return fmt.Errorf("failed to copy file %s: %w", file.Name, err)
		}
	}

	return nil
}

// updateOPFMetadata updates the metadata in the OPF file
func (e *EPUBEditor) updateOPFMetadata(extractDir string) error {
	// Find OPF file
	containerPath := filepath.Join(extractDir, "META-INF", "container.xml")
	containerContent, err := os.ReadFile(containerPath)
	if err != nil {
		return fmt.Errorf("failed to read container.xml: %w", err)
	}

	type Container struct {
		Rootfiles struct {
			Rootfile []struct {
				FullPath string `xml:"full-path,attr"`
			} `xml:"rootfile"`
		} `xml:"rootfiles"`
	}

	var container Container
	if err := xml.Unmarshal(containerContent, &container); err != nil {
		return fmt.Errorf("failed to parse container.xml: %w", err)
	}

	if len(container.Rootfiles.Rootfile) == 0 {
		return fmt.Errorf("no rootfile found in container.xml")
	}

	opfPath := filepath.Join(extractDir, container.Rootfiles.Rootfile[0].FullPath)
	opfContent, err := os.ReadFile(opfPath)
	if err != nil {
		return fmt.Errorf("failed to read OPF file: %w", err)
	}

	// Update metadata in OPF content
	updatedOPF, err := e.updateOPFContent(opfContent)
	if err != nil {
		return fmt.Errorf("failed to update OPF content: %w", err)
	}

	// Write back to file
	if err := os.WriteFile(opfPath, updatedOPF, 0644); err != nil {
		return fmt.Errorf("failed to write updated OPF file: %w", err)
	}

	return nil
}

// updateOPFContent updates the metadata within OPF XML content
func (e *EPUBEditor) updateOPFContent(opfContent []byte) ([]byte, error) {
	// Parse OPF
	opfStr := string(opfContent)

	// Update title
	opfStr = e.replaceXMLElement(opfStr, "dc:title", e.metadata.Title)

	// Update creator/author
	opfStr = e.replaceXMLElement(opfStr, "dc:creator", e.metadata.Author)

	// Update description
	if e.metadata.Description != "" {
		opfStr = e.replaceXMLElement(opfStr, "dc:description", e.metadata.Description)
	}

	// Update language
	if e.metadata.Language != "" {
		opfStr = e.replaceXMLElement(opfStr, "dc:language", e.metadata.Language)
	}

	// Update publisher
	if e.metadata.Publisher != "" {
		opfStr = e.replaceXMLElement(opfStr, "dc:publisher", e.metadata.Publisher)
	}

	// Update modified timestamp
	modifiedTime := time.Now().Format(time.RFC3339)
	opfStr = e.replaceMetaProperty(opfStr, "dcterms:modified", modifiedTime)

	return []byte(opfStr), nil
}

// replaceXMLElement replaces the content of an XML element
func (e *EPUBEditor) replaceXMLElement(content, element, newValue string) string {
	// Find the opening tag (with possible attributes)
	startPattern := fmt.Sprintf(`<%s`, element)
	startIdx := strings.Index(content, startPattern)
	if startIdx == -1 {
		return content
	}

	// Find the end of the opening tag
	tagEndIdx := strings.Index(content[startIdx:], ">")
	if tagEndIdx == -1 {
		return content
	}
	tagEndIdx += startIdx + 1

	// Find the closing tag
	endPattern := fmt.Sprintf(`</%s>`, element)
	endIdx := strings.Index(content[tagEndIdx:], endPattern)
	if endIdx == -1 {
		return content
	}
	endIdx += tagEndIdx

	// Replace the content between tags
	before := content[:tagEndIdx]
	after := content[endIdx:]

	return before + newValue + after
}

// replaceMetaProperty replaces the content of a meta property
func (e *EPUBEditor) replaceMetaProperty(content, property, newValue string) string {
	pattern := fmt.Sprintf(`property="%s"`, property)
	if strings.Contains(content, pattern) {
		// Find the meta tag with this property
		startIdx := strings.LastIndex(content[:strings.Index(content, pattern)], "<meta")
		if startIdx != -1 {
			endIdx := strings.Index(content[startIdx:], "/>")
			if endIdx != -1 {
				endIdx += startIdx + 2
				// Replace or add the content
				newMetaTag := fmt.Sprintf(`<meta property="%s">%s</meta>`, property, newValue)
				return content[:startIdx] + newMetaTag + content[endIdx:]
			}
		}
	}
	return content
}

// updateCoverImage updates the cover image in the EPUB
func (e *EPUBEditor) updateCoverImage(extractDir string) error {
	// Copy cover image to images directory
	imagesDir := filepath.Join(extractDir, "EPUB", "images")
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		return fmt.Errorf("failed to create images directory: %w", err)
	}

	coverExt := filepath.Ext(e.newCover)
	destPath := filepath.Join(imagesDir, "cover"+coverExt)

	return copyFile(e.newCover, destPath)
}

// repackageEPUB creates a new EPUB file from the extracted directory
func (e *EPUBEditor) repackageEPUB(extractDir, outputPath string) error {
	// Create ZIP file
	zipFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Add mimetype first (uncompressed)
	mimetypePath := filepath.Join(extractDir, "mimetype")
	if _, err := os.Stat(mimetypePath); err == nil {
		mimetypeData, err := os.ReadFile(mimetypePath)
		if err != nil {
			return fmt.Errorf("failed to read mimetype: %w", err)
		}

		w, err := zipWriter.CreateHeader(&zip.FileHeader{
			Name:   "mimetype",
			Method: zip.Store, // No compression
		})
		if err != nil {
			return fmt.Errorf("failed to create mimetype entry: %w", err)
		}

		if _, err := w.Write(mimetypeData); err != nil {
			return fmt.Errorf("failed to write mimetype: %w", err)
		}
	}

	// Add all other files
	return filepath.Walk(extractDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip mimetype (already added) and directories
		relPath, err := filepath.Rel(extractDir, path)
		if err != nil {
			return err
		}

		if relPath == "mimetype" || info.IsDir() {
			return nil
		}

		// Create ZIP entry
		w, err := zipWriter.Create(relPath)
		if err != nil {
			return fmt.Errorf("failed to create ZIP entry for %s: %w", relPath, err)
		}

		// Copy file content
		fileData, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		if _, err := w.Write(fileData); err != nil {
			return fmt.Errorf("failed to write file %s to ZIP: %w", relPath, err)
		}

		return nil
	})
}
