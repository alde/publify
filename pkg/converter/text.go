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

		if chapterPattern.MatchString(line) {
			if len(processed) > 0 && processed[len(processed)-1] != "" {
				processed = append(processed, "")
			}
			processed = append(processed, line)
			processed = append(processed, "")
			continue
		}

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

	if strings.ToUpper(line) == line && len(line) > 5 {
		return true
	}

	chapterPattern := regexp.MustCompile(`^(Chapter|CHAPTER|Ch\.|CH\.)\s*\d+`)
	return chapterPattern.MatchString(line)
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