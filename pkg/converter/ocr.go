package converter

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type OCRProcessor struct {
	language string
}

type OCRResult struct {
	Text       string
	Confidence int
	WordCount  int
	CharCount  int
}

func NewOCRProcessor(language string) (*OCRProcessor, error) {
	if !IsOCRAvailable() {
		return nil, fmt.Errorf("tesseract not available")
	}

	return &OCRProcessor{
		language: language,
	}, nil
}

func (ocr *OCRProcessor) ExtractTextFromImage(img image.Image) (string, error) {
	tempFile, err := ocr.saveImageToTemp(img)
	if err != nil {
		return "", fmt.Errorf("failed to save image to temp file: %w", err)
	}
	defer os.Remove(tempFile)

	return ocr.ExtractTextFromFile(tempFile)
}

func (ocr *OCRProcessor) ExtractTextFromFile(imagePath string) (string, error) {
	cmd := exec.Command("tesseract", imagePath, "stdout", "-l", ocr.language)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("OCR text extraction failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

func (ocr *OCRProcessor) ExtractTextWithStats(img image.Image) (OCRResult, error) {
	text, err := ocr.ExtractTextFromImage(img)
	if err != nil {
		return OCRResult{}, err
	}

	words := strings.Fields(text)

	return OCRResult{
		Text:       text,
		Confidence: 0, // Not available with direct binary call
		WordCount:  len(words),
		CharCount:  len(text),
	}, nil
}

func (ocr *OCRProcessor) saveImageToTemp(img image.Image) (string, error) {
	tempFile, err := os.CreateTemp("", "publify-ocr-*.png")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	if err := png.Encode(tempFile, img); err != nil {
		os.Remove(tempFile.Name())
		return "", err
	}

	return tempFile.Name(), nil
}

func (ocr *OCRProcessor) Close() error {
	return nil
}

func IsOCRAvailable() bool {
	_, err := exec.LookPath("tesseract")
	return err == nil
}

func (ocr *OCRProcessor) ProcessImageFile(imagePath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(imagePath))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".tiff", ".bmp":
		return ocr.ExtractTextFromFile(imagePath)
	default:
		return "", fmt.Errorf("unsupported image format: %s", ext)
	}
}