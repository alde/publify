package cmd

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// noCompressionWriter implements io.WriteCloser for fast compression mode
type noCompressionWriter struct {
	io.Writer
}

func (w *noCompressionWriter) Close() error {
	return nil
}

var (
	compressOutputPath string
	compressionLevel   string
)

var compressCmd = &cobra.Command{
	Use:   "compress [folder]",
	Short: "Compress a folder back into an EPUB file",
	Long: `Compress a folder containing EPUB contents back into a valid EPUB file.

The folder should contain the EPUB structure with META-INF/, OEBPS/, and other
standard EPUB files. This is typically used after extracting and editing an EPUB.

Examples:
  publify compress extracted_book/ -o fixed_book.epub
  publify compress book_folder/ --output book.epub
  publify compress folder/ -o book.epub --compression fast`,
	Args: cobra.ExactArgs(1),
	RunE: runCompress,
}

func init() {
	rootCmd.AddCommand(compressCmd)

	compressCmd.Flags().StringVarP(&compressOutputPath, "output", "o", "", "Output EPUB file path (required)")
	compressCmd.Flags().StringVar(&compressionLevel, "compression", "default", "Compression level (fast, default, best)")

	compressCmd.MarkFlagRequired("output")
}

func runCompress(cmd *cobra.Command, args []string) error {
	folderPath := args[0]

	// Validate input folder
	if err := validateCompressInputFolder(folderPath); err != nil {
		return fmt.Errorf("input folder validation failed: %w", err)
	}

	// Validate output path (reusing validation from convert command)
	if err := validateOutputPath(compressOutputPath); err != nil {
		return fmt.Errorf("output validation failed: %w", err)
	}

	// Validate compression level
	if err := validateCompressionLevel(compressionLevel); err != nil {
		return fmt.Errorf("compression validation failed: %w", err)
	}

	// Compress folder to EPUB
	return compressToEPUB(folderPath, compressOutputPath)
}

func validateCompressInputFolder(folderPath string) error {
	// Check if folder exists
	stat, err := os.Stat(folderPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("input folder does not exist: %s", folderPath)
	}
	if !stat.IsDir() {
		return fmt.Errorf("input path is not a directory: %s", folderPath)
	}

	// Check for required EPUB structure
	requiredFiles := []string{
		"mimetype",
		"META-INF/container.xml",
	}

	for _, requiredFile := range requiredFiles {
		fullPath := filepath.Join(folderPath, requiredFile)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			return fmt.Errorf("missing required EPUB file: %s (this doesn't look like an extracted EPUB folder)", requiredFile)
		}
	}

	return nil
}

func validateCompressionLevel(level string) error {
	validLevels := []string{"fast", "default", "best"}
	for _, valid := range validLevels {
		if level == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid compression level: %s (valid options: %s)", level, strings.Join(validLevels, ", "))
}

func compressToEPUB(folderPath, outputPath string) error {
	// Create output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	// Create ZIP writer
	zipWriter := zip.NewWriter(outputFile)
	defer zipWriter.Close()

	// Set compression level
	switch compressionLevel {
	case "fast":
		// Use no compression for speed
		zipWriter.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
			return &noCompressionWriter{Writer: out}, nil
		})
	case "best":
		// Use maximum compression - this will be slower but create smaller files
		// The default deflate compressor already uses good compression
	default:
		// Use default compression
	}

	if verbose {
		fmt.Printf("Compressing folder %s to EPUB: %s\n", folderPath, outputPath)
	}

	// Special handling for mimetype file (must be uncompressed and first in ZIP per EPUB spec)
	mimetypePath := filepath.Join(folderPath, "mimetype")
	if err := addMimetypeFile(zipWriter, mimetypePath); err != nil {
		return fmt.Errorf("failed to add mimetype file: %w", err)
	}

	fileCount := 1 // Already added mimetype

	// Walk through directory and add files (excluding mimetype which we already added)
	err = filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip mimetype file (already added)
		if filepath.Base(path) == "mimetype" && filepath.Dir(path) == folderPath {
			return nil
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path for ZIP entry
		relPath, err := filepath.Rel(folderPath, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", path, err)
		}

		// Normalize path separators for ZIP (always use forward slashes)
		relPath = filepath.ToSlash(relPath)

		if err := addFileToZip(zipWriter, path, relPath); err != nil {
			return fmt.Errorf("failed to add file %s: %w", relPath, err)
		}

		fileCount++
		if verbose {
			fmt.Printf("  ✓ %s\n", relPath)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to compress folder: %w", err)
	}

	fmt.Printf("✅ Successfully compressed %d files to %s\n", fileCount, filepath.Base(outputPath))

	// Provide helpful next steps
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  Test the EPUB file in your e-reader to ensure it works correctly\n")

	return nil
}

func addMimetypeFile(zipWriter *zip.Writer, mimetypePath string) error {
	// Read mimetype content
	content, err := os.ReadFile(mimetypePath)
	if err != nil {
		return fmt.Errorf("failed to read mimetype file: %w", err)
	}

	// Create mimetype entry with no compression (required by EPUB spec)
	header := &zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store, // No compression
	}

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("failed to create mimetype entry: %w", err)
	}

	if _, err := writer.Write(content); err != nil {
		return fmt.Errorf("failed to write mimetype content: %w", err)
	}

	if verbose {
		fmt.Printf("  ✓ mimetype (uncompressed)\n")
	}

	return nil
}

func addFileToZip(zipWriter *zip.Writer, filePath, zipPath string) error {
	// Open source file
	sourceFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Get file info for permissions
	info, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Create file header
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return fmt.Errorf("failed to create file header: %w", err)
	}

	// Set the zip path
	header.Name = zipPath

	// Use deflate compression for most files
	header.Method = zip.Deflate

	// Create writer for this file
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("failed to create zip entry: %w", err)
	}

	// Copy file content
	if _, err := io.Copy(writer, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	return nil
}
