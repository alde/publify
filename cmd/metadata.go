package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alde/publify/pkg/metadata"
	"github.com/spf13/cobra"
)

var (
	metaTitle       string
	metaAuthor      string
	metaDescription string
	metaLanguage    string
	metaPublisher   string
	metaCover       string
	showMeta        bool
)

var metadataCmd = &cobra.Command{
	Use:   "metadata [epub file]",
	Short: "View or edit EPUB metadata",
	Long: `View or edit metadata for EPUB files.

View metadata:
  publify metadata book.epub

Edit metadata:
  publify metadata book.epub --title "New Title" --author "New Author"
  publify metadata book.epub --description "Book description"
  publify metadata book.epub --cover cover.jpg

All metadata fields:
  --title       Book title
  --author      Author name
  --description Book description
  --language    Language code (e.g., en, sv, de)
  --publisher   Publisher name
  --cover       Path to cover image file`,
	Args: cobra.ExactArgs(1),
	RunE: runMetadata,
}

func init() {
	rootCmd.AddCommand(metadataCmd)

	metadataCmd.Flags().StringVar(&metaTitle, "title", "", "Set book title")
	metadataCmd.Flags().StringVar(&metaAuthor, "author", "", "Set author name")
	metadataCmd.Flags().StringVar(&metaDescription, "description", "", "Set book description")
	metadataCmd.Flags().StringVar(&metaLanguage, "language", "", "Set language code (e.g., en, sv)")
	metadataCmd.Flags().StringVar(&metaPublisher, "publisher", "", "Set publisher name")
	metadataCmd.Flags().StringVar(&metaCover, "cover", "", "Set cover image (path to image file)")
	metadataCmd.Flags().BoolVar(&showMeta, "show", false, "Show current metadata (default if no flags)")
}

func runMetadata(cmd *cobra.Command, args []string) error {
	epubPath := args[0]

	// Validate EPUB file
	if err := validateEPUBFile(epubPath); err != nil {
		return fmt.Errorf("EPUB validation failed: %w", err)
	}

	// Check if we're only viewing metadata
	if isViewOnlyMode() {
		return showMetadata(epubPath)
	}

	// Edit metadata
	return editMetadata(epubPath)
}

func validateEPUBFile(path string) error {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("EPUB file does not exist: %s", path)
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".epub" {
		return fmt.Errorf("file is not an EPUB: %s (expected .epub extension)", ext)
	}

	return nil
}

func isViewOnlyMode() bool {
	// If no editing flags are set, we're in view mode
	return metaTitle == "" &&
		metaAuthor == "" &&
		metaDescription == "" &&
		metaLanguage == "" &&
		metaPublisher == "" &&
		metaCover == ""
}

func showMetadata(epubPath string) error {
	reader, err := metadata.NewEPUBReader(epubPath)
	if err != nil {
		return fmt.Errorf("failed to open EPUB: %w", err)
	}
	defer reader.Close()

	meta, err := reader.GetMetadata()
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	// Display metadata in a nice format
	fmt.Printf("📖 EPUB Metadata: %s\n", filepath.Base(epubPath))
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	if meta.Title != "" {
		fmt.Printf("📝 Title:       %s\n", meta.Title)
	}
	if meta.Author != "" {
		fmt.Printf("✍️  Author:      %s\n", meta.Author)
	}
	if meta.Description != "" {
		fmt.Printf("📄 Description: %s\n", truncateText(meta.Description, 80))
	}
	if meta.Language != "" {
		fmt.Printf("🌍 Language:    %s\n", meta.Language)
	}
	if meta.Publisher != "" {
		fmt.Printf("🏢 Publisher:   %s\n", meta.Publisher)
	}
	if meta.Identifier != "" {
		fmt.Printf("🔗 Identifier:  %s\n", meta.Identifier)
	}
	if !meta.Created.IsZero() {
		fmt.Printf("📅 Created:     %s\n", meta.Created.Format("2006-01-02 15:04:05"))
	}
	if !meta.Modified.IsZero() {
		fmt.Printf("📝 Modified:    %s\n", meta.Modified.Format("2006-01-02 15:04:05"))
	}

	// Show file info
	stat, err := os.Stat(epubPath)
	if err == nil {
		fmt.Printf("📊 File Size:   %s\n", formatFileSize(stat.Size()))
	}

	// Show chapter count if available
	chapters, err := reader.GetChapterList()
	if err == nil && len(chapters) > 0 {
		fmt.Printf("📚 Chapters:    %d\n", len(chapters))
	}

	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	return nil
}

func editMetadata(epubPath string) error {
	// Create backup
	backupPath := epubPath + ".backup"
	if err := copyFile(epubPath, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	editor, err := metadata.NewEPUBEditor(epubPath)
	if err != nil {
		return fmt.Errorf("failed to open EPUB for editing: %w", err)
	}
	defer editor.Close()

	// Apply metadata changes
	changes := 0

	if metaTitle != "" {
		if err := editor.SetTitle(metaTitle); err != nil {
			return fmt.Errorf("failed to set title: %w", err)
		}
		changes++
		if verbose {
			fmt.Printf("✅ Set title: %s\n", metaTitle)
		}
	}

	if metaAuthor != "" {
		if err := editor.SetAuthor(metaAuthor); err != nil {
			return fmt.Errorf("failed to set author: %w", err)
		}
		changes++
		if verbose {
			fmt.Printf("✅ Set author: %s\n", metaAuthor)
		}
	}

	if metaDescription != "" {
		if err := editor.SetDescription(metaDescription); err != nil {
			return fmt.Errorf("failed to set description: %w", err)
		}
		changes++
		if verbose {
			fmt.Printf("✅ Set description: %s\n", truncateText(metaDescription, 50))
		}
	}

	if metaLanguage != "" {
		if err := editor.SetLanguage(metaLanguage); err != nil {
			return fmt.Errorf("failed to set language: %w", err)
		}
		changes++
		if verbose {
			fmt.Printf("✅ Set language: %s\n", metaLanguage)
		}
	}

	if metaPublisher != "" {
		if err := editor.SetPublisher(metaPublisher); err != nil {
			return fmt.Errorf("failed to set publisher: %w", err)
		}
		changes++
		if verbose {
			fmt.Printf("✅ Set publisher: %s\n", metaPublisher)
		}
	}

	if metaCover != "" {
		if err := validateCoverImage(metaCover); err != nil {
			return fmt.Errorf("cover image validation failed: %w", err)
		}

		if err := editor.SetCover(metaCover); err != nil {
			return fmt.Errorf("failed to set cover: %w", err)
		}
		changes++
		if verbose {
			fmt.Printf("✅ Set cover: %s\n", filepath.Base(metaCover))
		}
	}

	if changes == 0 {
		fmt.Println("No metadata changes specified. Use --help to see available options.")
		return nil
	}

	// Save changes
	if err := editor.Save(); err != nil {
		return fmt.Errorf("failed to save changes: %w", err)
	}

	// Remove backup if successful
	if err := os.Remove(backupPath); err != nil {
		fmt.Printf("Warning: failed to remove backup file: %s\n", backupPath)
	}

	fmt.Printf("✅ Successfully updated %d metadata field(s) in %s\n", changes, filepath.Base(epubPath))

	return nil
}

func validateCoverImage(imagePath string) error {
	// Check if file exists
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return fmt.Errorf("cover image does not exist: %s", imagePath)
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(imagePath))
	validExtensions := []string{".jpg", ".jpeg", ".png", ".webp"}

	for _, validExt := range validExtensions {
		if ext == validExt {
			return nil
		}
	}

	return fmt.Errorf("unsupported image format: %s (supported: %v)", ext, validExtensions)
}

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

func truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength-3] + "..."
}

// formatFileSize is duplicated here for the metadata command
// In a real implementation, this would be moved to a shared utilities package
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
