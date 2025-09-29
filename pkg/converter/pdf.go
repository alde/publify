package converter

import (
	"context"
	"fmt"
	"image"
	"os"
	"strings"
	"time"

	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/webassembly"
	"github.com/alde/publify/internal/worker"
)

type PDFPage struct {
	Number    int
	Text      string
	Images    []image.Image
	Width     float64
	Height    float64
	HasText   bool
	HasImage  bool
	PageType  PageType
	ImageData []byte // Raw image data for image pages
}

type PDFProcessor struct {
	filePath       string
	pdfBytes       []byte
	imagePageRange *PageRangeSet
	pool           pdfium.Pool
	pageCount      int
	enableOCR      bool
	ocrProcessor   *OCRProcessor
}

func NewPDFProcessor(filePath, imagePageRangeStr string, enableOCR bool, ocrLanguage string) (*PDFProcessor, error) {
	imagePageRange, err := ParsePageRanges(imagePageRangeStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse image page ranges: %w", err)
	}

	pdfBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF file: %w", err)
	}

	pool, err := webassembly.Init(webassembly.Config{
		MinIdle:  1,
		MaxIdle:  2,
		MaxTotal: 4,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize PDFium: %w", err)
	}

	instance, err := pool.GetInstance(time.Second * 30)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to get PDFium instance: %w", err)
	}

	doc, err := instance.OpenDocument(&requests.OpenDocument{
		File: &pdfBytes,
	})
	if err != nil {
		instance.Close()
		pool.Close()
		return nil, fmt.Errorf("failed to open PDF document: %w", err)
	}

	pageCountResp, err := instance.FPDF_GetPageCount(&requests.FPDF_GetPageCount{
		Document: doc.Document,
	})
	if err != nil {
		instance.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: doc.Document})
		instance.Close()
		pool.Close()
		return nil, fmt.Errorf("failed to get page count: %w", err)
	}

	pageCount := pageCountResp.PageCount

	instance.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: doc.Document})
	instance.Close()

	var ocrProcessor *OCRProcessor
	if enableOCR {
		var err error
		ocrProcessor, err = NewOCRProcessor(ocrLanguage)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("failed to initialize OCR processor: %w", err)
		}
	}

	processor := &PDFProcessor{
		filePath:       filePath,
		pdfBytes:       pdfBytes,
		imagePageRange: imagePageRange,
		pool:           pool,
		pageCount:      pageCount,
		enableOCR:      enableOCR,
		ocrProcessor:   ocrProcessor,
	}

	if imagePageRange != nil {
		if err := imagePageRange.ValidateAgainstTotal(pageCount); err != nil {
			processor.Close()
			return nil, fmt.Errorf("invalid page range: %w", err)
		}
	}

	return processor, nil
}

func (p *PDFProcessor) GetPageCount() int {
	return p.pageCount
}

func (p *PDFProcessor) ProcessPages(ctx context.Context, pool *worker.Pool, progressCallback func(int, int)) ([]PDFPage, error) {
	return p.processSequentially(ctx, progressCallback)
}

func (p *PDFProcessor) processSequentially(ctx context.Context, progressCallback func(int, int)) ([]PDFPage, error) {
	pageCount := p.GetPageCount()
	pages := make([]PDFPage, pageCount)

	for i := 0; i < pageCount; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		page, err := p.ProcessPage(i + 1)
		if err != nil {
			return nil, fmt.Errorf("failed to process page %d: %w", i+1, err)
		}

		pages[i] = page

		if progressCallback != nil {
			progressCallback(i+1, pageCount)
		}
	}

	return pages, nil
}

func (p *PDFProcessor) ProcessPage(pageNum int) (PDFPage, error) {
	if pageNum < 1 || pageNum > p.GetPageCount() {
		return PDFPage{}, fmt.Errorf("page number %d out of range (1-%d)", pageNum, p.GetPageCount())
	}

	instance, err := p.pool.GetInstance(time.Second * 30)
	if err != nil {
		return PDFPage{}, fmt.Errorf("failed to get PDFium instance: %w", err)
	}
	defer instance.Close()

	doc, err := instance.OpenDocument(&requests.OpenDocument{
		File: &p.pdfBytes,
	})
	if err != nil {
		return PDFPage{}, fmt.Errorf("failed to open PDF document: %w", err)
	}
	defer instance.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: doc.Document})

	pageType := GetPageType(pageNum, p.imagePageRange)

	pdfPage := PDFPage{
		Number:   pageNum,
		PageType: pageType,
		Width:    612.0,
		Height:   792.0,
	}

	pageText, err := instance.GetPageText(&requests.GetPageText{
		Page: requests.Page{
			ByIndex: &requests.PageByIndex{
				Document: doc.Document,
				Index:    pageNum - 1,
			},
		},
	})

	var text string
	if err != nil || pageText.Text == "" {
		text = ""
	} else {
		text = cleanText(pageText.Text)
	}

	// If text extraction failed or returned minimal text, try OCR
	shouldTryOCR := p.enableOCR && p.ocrProcessor != nil &&
		(text == "" || len(strings.TrimSpace(text)) < 50) // More reasonable threshold

	if shouldTryOCR {
		pageImage, err := instance.RenderPageInDPI(&requests.RenderPageInDPI{
			Page: requests.Page{
				ByIndex: &requests.PageByIndex{
					Document: doc.Document,
					Index:    pageNum - 1,
				},
			},
			DPI: 300,
		})
		if err == nil && pageImage.Image != nil {
			// Try OCR and use it if it provides significantly more text
			ocrText, ocrErr := p.ocrProcessor.ExtractTextFromImage(pageImage.Image)
			if ocrErr == nil {
				ocrTextClean := strings.TrimSpace(ocrText)
				textClean := strings.TrimSpace(text)

				// Use OCR if it provides more substantial text
				if len(ocrTextClean) > len(textClean)+20 || (textClean == "" && len(ocrTextClean) > 10) {
					text = ocrText
				}
			}
		}
	}

	pdfPage.Text = text
	pdfPage.HasText = len(strings.TrimSpace(text)) > 0

	if pageType == PageTypeImage {
		pdfPage.HasImage = true
	}

	return pdfPage, nil
}

func cleanText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	lines := strings.Split(text, "\n")
	var cleanLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanLines = append(cleanLines, line)
		} else if len(cleanLines) > 0 && cleanLines[len(cleanLines)-1] != "" {
			cleanLines = append(cleanLines, "")
		}
	}

	return strings.Join(cleanLines, "\n")
}

func (p *PDFProcessor) GetFileSize() (int64, error) {
	return int64(len(p.pdfBytes)), nil
}

func (p *PDFProcessor) Close() error {
	if p.pool != nil {
		p.pool.Close()
	}
	return nil
}