package cmd

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	extractOutputDir  string
	preserveStructure bool
)

var extractCmd = &cobra.Command{
	Use:   "extract [epub file]",
	Short: "Extract EPUB contents to a folder",
	Long: `Extract the contents of an EPUB file to a folder for manual editing.

The extracted folder will contain all EPUB files in their original structure,
including META-INF/, OEBPS/, and all content files.

Examples:
  publify extract book.epub -o extracted/
  publify extract book.epub --output book_extracted/
  publify extract book.epub --preserve-structure`,
	Args: cobra.ExactArgs(1),
	RunE: runExtract,
}

func init() {
	rootCmd.AddCommand(extractCmd)

	extractCmd.Flags().StringVarP(&extractOutputDir, "output", "o", "", "Output directory for extracted files (required)")
	extractCmd.Flags().BoolVar(&preserveStructure, "preserve-structure", true, "Preserve original EPUB directory structure")

	extractCmd.MarkFlagRequired("output")
}

func runExtract(cmd *cobra.Command, args []string) error {
	epubPath := args[0]

	// Validate EPUB file (reusing validation from metadata command)
	if err := validateEPUBFile(epubPath); err != nil {
		return fmt.Errorf("EPUB validation failed: %w", err)
	}

	// Validate output directory
	if err := validateExtractOutputDir(extractOutputDir); err != nil {
		return fmt.Errorf("output directory validation failed: %w", err)
	}

	// Extract EPUB
	return extractEPUB(epubPath, extractOutputDir)
}

func validateExtractOutputDir(outputDir string) error {
	// Check if directory already exists
	if _, err := os.Stat(outputDir); err == nil {
		return fmt.Errorf("output directory already exists: %s (choose a different path or remove existing directory)", outputDir)
	}

	// Check if parent directory exists
	parentDir := filepath.Dir(outputDir)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		return fmt.Errorf("parent directory does not exist: %s", parentDir)
	}

	return nil
}

func extractEPUB(epubPath, outputDir string) error {
	// Open EPUB file (which is a ZIP archive)
	zipReader, err := zip.OpenReader(epubPath)
	if err != nil {
		return fmt.Errorf("failed to open EPUB file: %w", err)
	}
	defer zipReader.Close()

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if verbose {
		fmt.Printf("Extracting EPUB to: %s\n", outputDir)
	}

	// Extract all files
	fileCount := 0
	for _, file := range zipReader.File {
		if err := extractFile(file, outputDir); err != nil {
			return fmt.Errorf("failed to extract file %s: %w", file.Name, err)
		}
		fileCount++

		if verbose {
			fmt.Printf("  ✓ %s\n", file.Name)
		}
	}

	fmt.Printf("✅ Successfully extracted %d files from %s to %s\n",
		fileCount, filepath.Base(epubPath), outputDir)

	// Provide helpful next steps
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  1. Edit files in %s as needed\n", outputDir)
	fmt.Printf("  2. Use 'publify compress %s -o new_book.epub' to create a new EPUB\n", outputDir)

	return nil
}

func extractFile(file *zip.File, destDir string) error {
	// Create the full destination path
	destPath := filepath.Join(destDir, file.Name)

	// Create directory if this is a directory entry
	if file.FileInfo().IsDir() {
		return os.MkdirAll(destPath, file.FileInfo().Mode())
	}

	// Create parent directories if they don't exist
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directories: %w", err)
	}

	// Open file in ZIP
	fileReader, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open file in ZIP: %w", err)
	}
	defer fileReader.Close()

	// Create destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy content
	if _, err := io.Copy(destFile, fileReader); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// Set file permissions to match original (because permissions matter, even in Sweden)
	if err := destFile.Chmod(file.FileInfo().Mode()); err != nil {
		// Non-fatal error - just warn
		if verbose {
			fmt.Printf("Warning: failed to set permissions for %s: %v\n", destPath, err)
		}
	}

	return nil
}
