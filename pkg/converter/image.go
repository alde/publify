package converter

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/chai2010/webp"
	"github.com/alde/publify/pkg/reader"
)

// ImageProcessor handles image optimization for e-readers
type ImageProcessor struct {
	profile reader.Profile
	tempDir string
}

// NewImageProcessor creates a new image processor
func NewImageProcessor(profile reader.Profile, tempDir string) *ImageProcessor {
	return &ImageProcessor{
		profile: profile,
		tempDir: tempDir,
	}
}

// ProcessImage optimizes an image for the target reader
func (ip *ImageProcessor) ProcessImage(inputPath string) (string, error) {
	// Open the original image
	img, err := imaging.Open(inputPath)
	if err != nil {
		return "", fmt.Errorf("failed to open image: %w", err)
	}

	// Get optimal processing settings
	settings := ip.profile.ImageProcessingSettings()

	// Resize if needed
	img = ip.resizeImage(img, settings)

	// Convert to grayscale if needed
	if settings.Grayscale {
		img = imaging.Grayscale(img)
	}

	// Determine output format
	outputFormat := ip.selectOptimalFormat(settings)

	// Generate output filename
	outputPath := ip.generateOutputPath(inputPath, outputFormat)

	// Save optimized image
	if err := ip.saveImage(img, outputPath, outputFormat, settings); err != nil {
		return "", fmt.Errorf("failed to save optimized image: %w", err)
	}

	return outputPath, nil
}

// resizeImage resizes an image to fit reader constraints
func (ip *ImageProcessor) resizeImage(img image.Image, settings reader.ImageSettings) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Check if resizing is needed
	if width <= settings.MaxWidth && height <= settings.MaxHeight {
		return img
	}

	// Calculate new dimensions maintaining aspect ratio
	ratio := float64(width) / float64(height)

	var newWidth, newHeight int
	if ratio > float64(settings.MaxWidth)/float64(settings.MaxHeight) {
		// Width is the limiting factor
		newWidth = settings.MaxWidth
		newHeight = int(float64(settings.MaxWidth) / ratio)
	} else {
		// Height is the limiting factor
		newHeight = settings.MaxHeight
		newWidth = int(float64(settings.MaxHeight) * ratio)
	}

	// Use high-quality resampling
	return imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)
}

// selectOptimalFormat chooses the best image format for the reader
func (ip *ImageProcessor) selectOptimalFormat(settings reader.ImageSettings) string {
	// Check if reader supports WebP (best compression)
	for _, format := range ip.profile.Capabilities.SupportedImageFormats {
		if format == "webp" {
			return "webp"
		}
	}

	// Fall back to preferred format
	return settings.Format
}

// generateOutputPath creates an output path for the processed image
func (ip *ImageProcessor) generateOutputPath(inputPath, format string) string {
	base := filepath.Base(inputPath)
	name := strings.TrimSuffix(base, filepath.Ext(base))

	var ext string
	switch format {
	case "webp":
		ext = ".webp"
	case "png":
		ext = ".png"
	default:
		ext = ".jpg"
	}

	return filepath.Join(ip.tempDir, name+"_optimized"+ext)
}

// saveImage saves an image in the specified format with optimization
func (ip *ImageProcessor) saveImage(img image.Image, outputPath, format string, settings reader.ImageSettings) error {
	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	switch format {
	case "webp":
		return ip.saveAsWebP(img, outFile, settings)

	case "png":
		return ip.saveAsPNG(img, outFile, settings)

	default: // JPEG
		return ip.saveAsJPEG(img, outFile, settings)
	}
}

// saveAsJPEG saves an image as JPEG with specified quality
func (ip *ImageProcessor) saveAsJPEG(img image.Image, file *os.File, settings reader.ImageSettings) error {
	quality := settings.Quality

	// Adjust quality based on compression level
	if ip.profile.Capabilities.AggressiveCompression {
		switch settings.CompressionLevel {
		case "high":
			quality = min(quality, 75) // Very aggressive for file size
		case "medium":
			quality = min(quality, 85)
		}
	}

	options := &jpeg.Options{Quality: quality}
	return jpeg.Encode(file, img, options)
}

// saveAsWebP saves an image as WebP with high compression
func (ip *ImageProcessor) saveAsWebP(img image.Image, file *os.File, settings reader.ImageSettings) error {
	quality := float32(settings.Quality)

	// Adjust quality for WebP - it's more efficient so we can use higher values
	if ip.profile.Capabilities.AggressiveCompression {
		switch settings.CompressionLevel {
		case "high":
			quality = 70 // Very aggressive for WebP
		case "medium":
			quality = 80
		default:
			quality = 75
		}
	}

	// WebP quality is 0-100, same as JPEG
	options := &webp.Options{
		Lossless: false,
		Quality:  quality,
	}

	return webp.Encode(file, img, options)
}

// saveAsPNG saves an image as PNG
func (ip *ImageProcessor) saveAsPNG(img image.Image, file *os.File, settings reader.ImageSettings) error {
	encoder := &png.Encoder{
		CompressionLevel: png.BestCompression, // Always use best compression for file size
	}
	return encoder.Encode(file, img)
}

// GetOptimizedSize estimates the size reduction from optimization
func (ip *ImageProcessor) GetOptimizedSize(inputPath string) (int64, int64, error) {
	// Get original size
	originalStat, err := os.Stat(inputPath)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to stat original image: %w", err)
	}
	originalSize := originalStat.Size()

	// Process image
	optimizedPath, err := ip.ProcessImage(inputPath)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to process image: %w", err)
	}

	// Get optimized size
	optimizedStat, err := os.Stat(optimizedPath)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to stat optimized image: %w", err)
	}
	optimizedSize := optimizedStat.Size()

	return originalSize, optimizedSize, nil
}

// CleanupTempFiles removes temporary image files
func (ip *ImageProcessor) CleanupTempFiles() error {
	if ip.tempDir == "" {
		return nil
	}
	return os.RemoveAll(ip.tempDir)
}

// ImageStats contains statistics about image processing
type ImageStats struct {
	TotalImages     int
	ProcessedImages int
	OriginalSize    int64
	OptimizedSize   int64
	CompressionRatio float64
}

// CalculateImageStats calculates compression statistics
func (ip *ImageProcessor) CalculateImageStats(originalSizes, optimizedSizes []int64) ImageStats {
	var totalOriginal, totalOptimized int64

	for _, size := range originalSizes {
		totalOriginal += size
	}

	for _, size := range optimizedSizes {
		totalOptimized += size
	}

	ratio := 0.0
	if totalOriginal > 0 {
		ratio = float64(totalOptimized) / float64(totalOriginal)
	}

	return ImageStats{
		TotalImages:      len(originalSizes),
		ProcessedImages:  len(optimizedSizes),
		OriginalSize:     totalOriginal,
		OptimizedSize:    totalOptimized,
		CompressionRatio: ratio,
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}