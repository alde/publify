package converter

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// RepairPDF attempts to fix common PDF issues like missing or corrupted EOF
func RepairPDF(inputPath string) (string, error) {
	// Read the entire file
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return "", fmt.Errorf("failed to read PDF file: %w", err)
	}

	// Convert to string for easier manipulation
	content := string(data)

	// Find the last occurrence of %%EOF
	lastEOFIndex := strings.LastIndex(content, "%%EOF")
	if lastEOFIndex == -1 {
		return "", fmt.Errorf("PDF file does not contain %%EOF marker")
	}

	// Truncate everything after %%EOF + the marker itself (5 characters)
	// and add a proper line ending
	repairedContent := content[:lastEOFIndex+5] + "\n"

	// Create a temporary repaired file
	tempFile, err := os.CreateTemp("", "publify-repaired-*.pdf")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer tempFile.Close()

	// Write the repaired content
	if _, err := tempFile.WriteString(repairedContent); err != nil {
		return "", fmt.Errorf("failed to write repaired PDF: %w", err)
	}

	return tempFile.Name(), nil
}

// CleanupTempFile removes a temporary file
func CleanupTempFile(path string) error {
	if path != "" {
		return os.Remove(path)
	}
	return nil
}

// ValidatePDF performs basic PDF validation
func ValidatePDF(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("cannot open file: %w", err)
	}
	defer file.Close()

	// Check PDF header
	header := make([]byte, 8)
	if _, err := file.Read(header); err != nil {
		return fmt.Errorf("cannot read PDF header: %w", err)
	}

	if !strings.HasPrefix(string(header), "%PDF-") {
		return fmt.Errorf("file does not start with PDF header")
	}

	// Check for EOF marker somewhere in the file
	file.Seek(-1024, io.SeekEnd) // Check last 1KB for EOF
	buffer := make([]byte, 1024)
	n, _ := file.Read(buffer)

	if !strings.Contains(string(buffer[:n]), "%%EOF") {
		return fmt.Errorf("PDF file does not contain %%EOF marker")
	}

	return nil
}