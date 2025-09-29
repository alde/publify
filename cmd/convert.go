package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/alde/publify/pkg/converter"
	"github.com/alde/publify/pkg/reader"
)

var (
	outputPath   string
	readerType   string
	enableColor  bool
	workerCount  int
	enableOCR    bool
	ocrLanguage  string
	imagePages   string
)

var convertCmd = &cobra.Command{
	Use:   "convert [input file]",
	Short: "Convert documents between formats",
	Long: `Convert documents between formats with reader-specific optimizations.

Currently supports:
- PDF to EPUB conversion

Examples:
  publify convert input.pdf -o output.epub --reader kobo --color
  publify convert book.pdf -o book.epub --reader kobo --image-pages "1-2,419-420"`,
	Args: cobra.ExactArgs(1),
	RunE: runConvert,
}

func init() {
	rootCmd.AddCommand(convertCmd)

	convertCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path (required)")
	convertCmd.Flags().StringVar(&readerType, "reader", "generic", "Target reader type (kobo, kindle, generic)")
	convertCmd.Flags().BoolVar(&enableColor, "color", false, "Enable color processing for color e-readers")
	convertCmd.Flags().IntVar(&workerCount, "workers", 0, "Number of worker goroutines (0 = auto)")
	convertCmd.Flags().BoolVar(&enableOCR, "ocr", false, "Enable OCR for scanned PDFs (requires Tesseract)")
	convertCmd.Flags().StringVar(&ocrLanguage, "ocr-lang", "eng", "OCR language (eng, sve, deu, etc.)")
	convertCmd.Flags().StringVar(&imagePages, "image-pages", "", "Page ranges to treat as images (e.g., \"1-2,419-420\")")

	convertCmd.MarkFlagRequired("output")
}

func runConvert(cmd *cobra.Command, args []string) error {
	inputPath := args[0]

	// Validate input file
	if err := validateInputFile(inputPath); err != nil {
		return fmt.Errorf("input validation failed: %w", err)
	}

	// Validate output path
	if err := validateOutputPath(outputPath); err != nil {
		return fmt.Errorf("output validation failed: %w", err)
	}

	// Get reader profile
	profile, err := reader.GetProfile(readerType)
	if err != nil {
		return fmt.Errorf("reader profile error: %w", err)
	}

	// Override color support if explicitly disabled
	if !enableColor {
		profile.Capabilities.SupportsColor = false
	}

	// Check OCR availability if requested
	if enableOCR && !converter.IsOCRAvailable() {
		return fmt.Errorf("OCR requested but Tesseract not available. Please install Tesseract OCR")
	}

	// Validate image pages format if provided
	if imagePages != "" {
		_, err := converter.ParsePageRanges(imagePages)
		if err != nil {
			return fmt.Errorf("invalid image pages format: %w", err)
		}
	}

	// Set up converter options
	opts := converter.Options{
		InputPath:      inputPath,
		OutputPath:     outputPath,
		Profile:        profile,
		WorkerCount:    workerCount,
		Verbose:        verbose,
		EnableOCR:      enableOCR,
		OCRLanguage:    ocrLanguage,
		ImagePageRange: imagePages,
	}

	// Run conversion
	conv := converter.New(opts)
	return conv.Convert()
}

func validateInputFile(path string) error {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", path)
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".pdf" {
		return fmt.Errorf("unsupported input format: %s (only .pdf is supported)", ext)
	}

	return nil
}

func validateOutputPath(path string) error {
	// Check if output directory exists
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("output directory does not exist: %s", dir)
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".epub" {
		return fmt.Errorf("unsupported output format: %s (only .epub is supported)", ext)
	}

	return nil
}

var verbose bool

func init() {
	convertCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}