package converter

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode"
)

type TextProcessor struct {
	options TextProcessingOptions
}

type TextProcessingOptions struct {
	PreserveFormatting bool // Whether to maintain original formatting
	MinimizeFileSize   bool // Optimize for smaller file size
	ConvertToHTML      bool // Convert to HTML markup
}

func NewTextProcessor(opts TextProcessingOptions) *TextProcessor {
	return &TextProcessor{
		options: opts,
	}
}

func (tp *TextProcessor) ProcessText(text string) string {
	if text == "" {
		return text
	}

	text = tp.basicCleanup(text)
	text = tp.removeBookArtifacts(text) // Remove headers, footers, page numbers
	text = tp.normalizeWhitespace(text)
	text = tp.processChapters(text)
	if tp.options.ConvertToHTML {
		text = tp.convertToHTML(text)
	}

	return text
}

func (tp *TextProcessor) basicCleanup(text string) string {
	text = strings.Map(func(r rune) rune {
		if r == 0 || (unicode.IsControl(r) && r != '\n' && r != '\t') {
			return -1
		}
		return r
	}, text)

	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}

	return strings.Join(lines, "\n")
}

// removeBookArtifacts removes common book formatting artifacts (headers, footers, page numbers)
func (tp *TextProcessor) removeBookArtifacts(text string) string {
	lines := strings.Split(text, "\n")
	var cleanLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			cleanLines = append(cleanLines, "")
			continue
		}

		// Skip if it's just a page number (common patterns)
		if tp.isPageNumber(line) {
			continue
		}

		// Skip if it's a book header/footer (book title, chapter name in header)
		if tp.isBookHeader(line) {
			continue
		}

		cleanLines = append(cleanLines, line)
	}

	return strings.Join(cleanLines, "\n")
}

// isPageNumber detects if a line is likely just a page number
func (tp *TextProcessor) isPageNumber(line string) bool {
	line = strings.TrimSpace(line)

	// Just digits (page numbers starting from page 11)
	if regexp.MustCompile(`^\d+$`).MatchString(line) {
		return true
	}

	// Page number with formatting like "- 42 -" or "42."
	if regexp.MustCompile(`^[-\s]*\d+[-\s\\.]*$`).MatchString(line) {
		return true
	}

	return false
}

// isBookHeader detects if a line is likely a book header/footer
func (tp *TextProcessor) isBookHeader(line string) bool {
	line = strings.TrimSpace(line)

	// Too short to be meaningful content
	if len(line) < 3 {
		return false
	}

	// Common book title patterns (like "Air Babylon" in cursive/italic markup)
	bookTitles := []string{
		"Air Babylon",
		"AIR BABYLON",
		"air babylon",
	}

	for _, title := range bookTitles {
		if strings.Contains(strings.ToLower(line), strings.ToLower(title)) {
			return true
		}
	}

	// Very short lines that are likely headers (chapter names, etc.)
	// But exclude time spans which could be chapter markers
	if len(line) <= 30 && !tp.isTimeSpan(line) {
		// Check if it's all caps (likely a header)
		if strings.ToUpper(line) == line && len(strings.Fields(line)) <= 3 {
			return true
		}
	}

	return false
}

// isTimeSpan detects time-based chapter markers like "5-6am" or "11pm-12am"
func (tp *TextProcessor) isTimeSpan(line string) bool {
	line = strings.ToLower(strings.TrimSpace(line))

	// Patterns like "5-6am", "11pm-12am", "2.30-3.30pm"
	timePatterns := []string{
		`^\d{1,2}(:\d{2})?(-|\s*to\s*)\d{1,2}(:\d{2})?(am|pm)$`,
		`^\d{1,2}(:\d{2})?(am|pm)(-|\s*to\s*)\d{1,2}(:\d{2})?(am|pm)$`,
		`^\d{1,2}\.\d{2}(-|\s*to\s*)\d{1,2}\.\d{2}(am|pm)$`,
	}

	for _, pattern := range timePatterns {
		if regexp.MustCompile(pattern).MatchString(line) {
			return true
		}
	}

	return false
}

func (tp *TextProcessor) normalizeWhitespace(text string) string {
	if tp.options.MinimizeFileSize {
		text = regexp.MustCompile(`[ \t]+`).ReplaceAllString(text, " ")
		text = regexp.MustCompile(`\n[ \t]*\n`).ReplaceAllString(text, "\n\n")
		text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")
	} else {
		text = regexp.MustCompile(`[ \t]+`).ReplaceAllString(text, " ")
		text = regexp.MustCompile(`\n[ \t]*\n[ \t]*\n`).ReplaceAllString(text, "\n\n")
	}

	return strings.TrimSpace(text)
}

func (tp *TextProcessor) processChapters(text string) string {
	lines := strings.Split(text, "\n")
	var processed []string

	chapterPattern := regexp.MustCompile(`^(Chapter|CHAPTER|Ch\.|CH\.)\s*\d+`)
	sectionPattern := regexp.MustCompile(`^[A-Z][A-Z\s]{10,}$`) // All caps headers

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" {
			processed = append(processed, "")
			continue
		}

		// Detect traditional chapter markers
		if chapterPattern.MatchString(line) {
			if len(processed) > 0 && processed[len(processed)-1] != "" {
				processed = append(processed, "")
			}
			processed = append(processed, line)
			processed = append(processed, "")
			continue
		}

		// Detect time-based chapter markers (like "5-6am")
		if tp.isTimeSpan(line) {
			if len(processed) > 0 && processed[len(processed)-1] != "" {
				processed = append(processed, "")
			}
			// Format the time span as a proper chapter heading
			processed = append(processed, line)
			processed = append(processed, "")
			continue
		}

		// Detect all-caps section headers
		if sectionPattern.MatchString(line) && len(line) < 100 {
			processed = append(processed, "")
			processed = append(processed, line)
			processed = append(processed, "")
			continue
		}

		processed = append(processed, line)
	}

	return strings.Join(processed, "\n")
}

func (tp *TextProcessor) convertToHTML(text string) string {
	text = html.EscapeString(text)

	lines := strings.Split(text, "\n")
	var htmlLines []string
	inParagraph := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" {
			if inParagraph {
				htmlLines = append(htmlLines, "</p>")
				inParagraph = false
			}
			continue
		}

		if tp.isHeader(line) {
			if inParagraph {
				htmlLines = append(htmlLines, "</p>")
				inParagraph = false
			}
			htmlLines = append(htmlLines, fmt.Sprintf("<h2>%s</h2>", line))
			continue
		}

		if !inParagraph {
			htmlLines = append(htmlLines, "<p>")
			inParagraph = true
		}

		htmlLines = append(htmlLines, line+"<br/>")
	}

	if inParagraph {
		htmlLines = append(htmlLines, "</p>")
	}

	return strings.Join(htmlLines, "\n")
}

func (tp *TextProcessor) isHeader(line string) bool {
	if len(line) > 100 {
		return false
	}

	// Traditional chapter patterns
	chapterPattern := regexp.MustCompile(`^(Chapter|CHAPTER|Ch\.|CH\.)\s*\d+`)
	if chapterPattern.MatchString(line) {
		return true
	}

	// Time-based chapter markers (like "5-6am")
	if tp.isTimeSpan(line) {
		return true
	}

	// All caps headers
	if strings.ToUpper(line) == line && len(line) > 5 {
		return true
	}

	return false
}

func (tp *TextProcessor) EstimateTextSize(text string) int {
	processed := tp.ProcessText(text)
	return len([]byte(processed))
}

func (tp *TextProcessor) SplitIntoChunks(text string, maxChunkSize int) []string {
	if len(text) <= maxChunkSize {
		return []string{text}
	}

	var chunks []string
	paragraphs := strings.Split(text, "\n\n")
	currentChunk := ""

	for _, paragraph := range paragraphs {
		if len(currentChunk)+len(paragraph)+2 > maxChunkSize {
			if currentChunk != "" {
				chunks = append(chunks, strings.TrimSpace(currentChunk))
				currentChunk = ""
			}

			if len(paragraph) > maxChunkSize {
				sentences := tp.splitBySentences(paragraph, maxChunkSize)
				chunks = append(chunks, sentences...)
			} else {
				currentChunk = paragraph
			}
		} else {
			if currentChunk != "" {
				currentChunk += "\n\n"
			}
			currentChunk += paragraph
		}
	}

	if currentChunk != "" {
		chunks = append(chunks, strings.TrimSpace(currentChunk))
	}

	return chunks
}

func (tp *TextProcessor) splitBySentences(paragraph string, maxSize int) []string {
	sentences := regexp.MustCompile(`[.!?]+\s+`).Split(paragraph, -1)
	var chunks []string
	currentChunk := ""

	for _, sentence := range sentences {
		if len(currentChunk)+len(sentence)+1 > maxSize {
			if currentChunk != "" {
				chunks = append(chunks, strings.TrimSpace(currentChunk))
				currentChunk = ""
			}

			if len(sentence) > maxSize {
				chunks = append(chunks, strings.TrimSpace(sentence))
			} else {
				currentChunk = sentence
			}
		} else {
			if currentChunk != "" {
				currentChunk += " "
			}
			currentChunk += sentence
		}
	}

	if currentChunk != "" {
		chunks = append(chunks, strings.TrimSpace(currentChunk))
	}

	return chunks
}
