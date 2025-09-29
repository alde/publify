package converter

import (
	"context"
	"fmt"
	"image"
	"math"
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
	markovChain    *MarkovChain
	skipPages      map[int]bool
	rejectedPages  []int // Pages that failed Markov chain validation
}

func NewPDFProcessor(filePath, imagePageRangeStr string, enableOCR bool, ocrLanguage string, skipPagesStr string) (*PDFProcessor, error) {
	imagePageRange, err := ParsePageRanges(imagePageRangeStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse image page ranges: %w", err)
	}

	// Parse skip pages
	skipPages, err := parseSkipPages(skipPagesStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse skip pages: %w", err)
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

	// Initialize Markov chain for bleed-through detection
	markovChain := NewEnglishMarkovChain()

	processor := &PDFProcessor{
		filePath:       filePath,
		pdfBytes:       pdfBytes,
		imagePageRange: imagePageRange,
		pool:           pool,
		pageCount:      pageCount,
		enableOCR:      enableOCR,
		ocrProcessor:   ocrProcessor,
		markovChain:    markovChain,
		skipPages:      skipPages,
		rejectedPages:  make([]int, 0),
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

	// Check if this page should be skipped
	if p.skipPages[pageNum] {
		return PDFPage{
			Number:   pageNum,
			Text:     "",
			HasText:  false,
			HasImage: false,
			PageType: PageTypeText,
			Width:    612.0,
			Height:   792.0,
		}, nil
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
		if err == nil && pageImage.Result.Image != nil {
			// Clean up the image when done
			defer pageImage.Cleanup()

			// Try OCR and use it if it provides significantly more text
			ocrText, ocrErr := p.ocrProcessor.ExtractTextFromImage(pageImage.Result.Image)
			if ocrErr == nil {
				ocrTextClean := strings.TrimSpace(ocrText)
				textClean := strings.TrimSpace(text)

				// Use OCR if it provides more substantial text, but avoid garbled bleed-through
				if len(ocrTextClean) > len(textClean)+20 || (textClean == "" && len(ocrTextClean) > 10) {
					// Check if OCR text looks like garbled bleed-through
					if !p.isLikelyBleedThrough(pageNum, ocrTextClean) {
						text = ocrText
					}
				}
			}
		}
	}

	// Also check regular extracted text for bleed-through patterns
	if text != "" && len(strings.TrimSpace(text)) >= 20 {
		if p.isLikelyBleedThrough(pageNum, text) {
			// If the text is bleed-through, clear it
			text = ""
		}
	}

	pdfPage.Text = text
	pdfPage.HasText = len(strings.TrimSpace(text)) > 0

	if pageType == PageTypeImage {
		pdfPage.HasImage = true
	}

	return pdfPage, nil
}

// parseSkipPages converts a comma-separated string of page numbers to a map
func parseSkipPages(skipPagesStr string) (map[int]bool, error) {
	skipPages := make(map[int]bool)

	if skipPagesStr == "" {
		return skipPages, nil
	}

	pageStrs := strings.Split(skipPagesStr, ",")
	for _, pageStr := range pageStrs {
		pageStr = strings.TrimSpace(pageStr)
		if pageStr == "" {
			continue
		}

		// Parse page number
		pageNum := 0
		for _, char := range pageStr {
			if char < '0' || char > '9' {
				return nil, fmt.Errorf("invalid page number: %s", pageStr)
			}
			pageNum = pageNum*10 + int(char-'0')
		}

		if pageNum <= 0 {
			return nil, fmt.Errorf("page number must be positive: %s", pageStr)
		}

		skipPages[pageNum] = true
	}

	return skipPages, nil
}

// MarkovChain represents a simple character-level Markov chain for English text
type MarkovChain struct {
	transitions map[string]map[rune]int
	totals      map[string]int
}

// NewEnglishMarkovChain creates a Markov chain trained on common English patterns
func NewEnglishMarkovChain() *MarkovChain {
	mc := &MarkovChain{
		transitions: make(map[string]map[rune]int),
		totals:      make(map[string]int),
	}

	// Train on comprehensive English patterns including British English
	englishSamples := []string{
		"the quick brown fox jumps over the lazy dog",
		"this is a sample of normal english text that should have good probability",
		"common words like and the with for not but his from they she her been than",
		"normal sentences with proper punctuation and capitalization",
		"reading writing speaking listening are important language skills",
		"chapter one introduction to the basic concepts of literature and writing",
		"the author presents compelling arguments about human nature and society",
		"in this section we examine the historical context and its implications",
		"furthermore the evidence suggests that these conclusions are well founded",
		"therefore it becomes clear that understanding these principles is essential",
		"however there are several important considerations that must be addressed",
		"consequently the reader should carefully evaluate these different perspectives",
		"meanwhile the protagonist discovers new information that changes everything",
		"nevertheless the fundamental questions remain unanswered and require further study",
		"although the initial results were promising the final outcome was disappointing",
		"because of these factors the committee decided to postpone the final decision",
		"according to recent research findings the phenomenon occurs more frequently than expected",
		"throughout history many scholars have attempted to explain this complex relationship",
		"during the investigation several witnesses provided contradictory statements about the events",
		"despite numerous attempts to resolve the conflict the parties could not reach agreement",
		// British English and aviation terms
		"the flight attendant recognised the problem straightaway and organised a proper response",
		"bloody hell that was brilliant absolutely smashing work mate well done indeed",
		"the aircraft taxied to the gate whilst passengers organised their belongings and waited patiently",
		"check in desk queue baggage handlers uniform security clearance airport terminal",
		"the crew realised they needed to prioritise safety whilst maintaining excellent customer service",
		"favourite colour honour neighbour centre theatre licence practise organised travelled cancelled",
		"brilliant chap lovely weather rather fancy spot of tea properly sorted cheers mate",
		"aeroplane petrol colour grey aluminium whilst amongst programme tyre plough labour favour",
		"flight crew cabin pressure oxygen masks emergency procedures safety demonstration boarding passes",
		"immigration customs duty free departure lounge boarding gate overhead compartments seat belts",
		"turbulence captain announcement weather conditions delayed cancelled diverted rescheduled",
		"first class business class economy premium seats upgrades frequent flyer miles loyalty points",
		"runway takeoff landing approach air traffic control tower ground staff maintenance hangar",
	}

	for _, sample := range englishSamples {
		mc.train(strings.ToLower(sample))
	}

	return mc
}

func (mc *MarkovChain) train(text string) {
	runes := []rune(text)
	for i := 0; i < len(runes)-1; i++ {
		current := string(runes[i])
		next := runes[i+1]

		if mc.transitions[current] == nil {
			mc.transitions[current] = make(map[rune]int)
		}

		mc.transitions[current][next]++
		mc.totals[current]++
	}
}

func (mc *MarkovChain) getTransitionProbability(current string, next rune) float64 {
	if total, exists := mc.totals[current]; exists && total > 0 {
		if count, exists := mc.transitions[current][next]; exists {
			return float64(count) / float64(total)
		}
	}
	return 0.01 // Small probability for unseen transitions
}

func (mc *MarkovChain) scoreText(text string) float64 {
	text = strings.ToLower(text)
	runes := []rune(text)
	if len(runes) < 2 {
		return -10.0
	}

	logProb := 0.0
	count := 0
	suspiciousPatterns := 0
	totalChars := 0

	for i := 0; i < len(runes)-1; i++ {
		current := string(runes[i])
		next := runes[i+1]

		// Count all characters for pattern analysis
		if runes[i] >= 'a' && runes[i] <= 'z' {
			totalChars++
		}

		// Only score alphabetic transitions
		if (runes[i] >= 'a' && runes[i] <= 'z') && (next >= 'a' && next <= 'z') {
			prob := mc.getTransitionProbability(current, next)
			logProb += math.Log(prob)
			count++

			// Check for suspicious patterns common in OCR bleed-through
			if prob < 0.005 { // Very unlikely transitions
				suspiciousPatterns++
			}
		}
	}

	if count == 0 {
		return -10.0 // Very low score for non-alphabetic text
	}

	// Base score from Markov chain
	baseScore := logProb / float64(count)

	// Apply penalty for high ratio of suspicious patterns
	suspiciousRatio := float64(suspiciousPatterns) / float64(count)
	suspiciousPenalty := suspiciousRatio * -2.0

	// Apply penalty for excessive single character occurrences (like "a: a: a:")
	singleCharPenalty := mc.calculateSingleCharPenalty(text)

	finalScore := baseScore + suspiciousPenalty + singleCharPenalty

	return finalScore
}

func (mc *MarkovChain) calculateSingleCharPenalty(text string) float64 {
	// Count patterns like repeated single characters or very short fragments
	words := strings.Fields(text)
	singleCharCount := 0
	totalWords := len(words)

	if totalWords == 0 {
		return -1.0
	}

	for _, word := range words {
		// Remove punctuation for analysis
		cleanWord := strings.Trim(word, ".,!?:;")
		if len(cleanWord) == 1 || len(cleanWord) == 2 {
			singleCharCount++
		}
	}

	// Penalty for high ratio of very short words (common in garbled text)
	shortWordRatio := float64(singleCharCount) / float64(totalWords)
	if shortWordRatio > 0.4 { // More than 40% single/double char words
		return -1.5
	} else if shortWordRatio > 0.2 { // More than 20% single/double char words
		return -0.5
	}

	return 0.0
}

// isLikelyBleedThrough detects OCR bleed-through using Markov chain analysis
func (p *PDFProcessor) isLikelyBleedThrough(pageNum int, text string) bool {
	text = strings.TrimSpace(text)
	if len(text) < 20 {
		return false
	}

	fmt.Printf("DEBUG: Analyzing text: '%.50s...'\n", text)

	// Use Markov chain to score the text
	score := p.markovChain.scoreText(text)
	fmt.Printf("DEBUG: Markov chain score: %.3f\n", score)

	// Typical scores with enhanced algorithm:
	// Real English text: around -1.5 to -2.5
	// Garbled OCR text: around -4.0 to -6.0 or worse
	threshold := -3.8

	isBleedThrough := score < threshold
	fmt.Printf("DEBUG: Score %.3f vs threshold %.3f, is bleed-through: %t\n", score, threshold, isBleedThrough)

	// Track pages that were rejected for post-conversion reporting
	if isBleedThrough {
		p.rejectedPages = append(p.rejectedPages, pageNum)
	}

	return isBleedThrough
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

// GetRejectedPages returns the list of pages that were rejected by Markov chain validation
func (p *PDFProcessor) GetRejectedPages() []int {
	return p.rejectedPages
}

// ValidateTextContent tests text content against the Markov chain bleed-through detection
func (p *PDFProcessor) ValidateTextContent(text string, threshold float64) (float64, bool) {
	if p.markovChain == nil {
		return 0.0, false
	}

	text = strings.TrimSpace(text)
	if len(text) < 20 {
		return 0.0, false
	}

	score := p.markovChain.scoreText(text)
	isBleedThrough := score < threshold

	return score, isBleedThrough
}