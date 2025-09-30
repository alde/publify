package converter

import (
	"fmt"
	"strconv"
	"strings"
)

// PageRange represents a range of pages
type PageRange struct {
	Start int
	End   int
}

// PageRangeSet holds multiple page ranges
type PageRangeSet struct {
	ranges []PageRange
}

// ParsePageRanges parses a page range string like "1-2,5,10-15,419-420"
func ParsePageRanges(rangeStr string) (*PageRangeSet, error) {
	if rangeStr == "" {
		return &PageRangeSet{}, nil
	}

	var ranges []PageRange
	parts := strings.Split(rangeStr, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "-") {
			// Range like "1-5"
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range format: %s", part)
			}

			start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid start page: %s", rangeParts[0])
			}

			end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid end page: %s", rangeParts[1])
			}

			if start > end {
				return nil, fmt.Errorf("start page (%d) cannot be greater than end page (%d)", start, end)
			}

			ranges = append(ranges, PageRange{Start: start, End: end})
		} else {
			// Single page like "5"
			page, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid page number: %s", part)
			}

			ranges = append(ranges, PageRange{Start: page, End: page})
		}
	}

	return &PageRangeSet{ranges: ranges}, nil
}

// Contains checks if a page number is within any of the ranges
func (prs *PageRangeSet) Contains(pageNum int) bool {
	for _, r := range prs.ranges {
		if pageNum >= r.Start && pageNum <= r.End {
			return true
		}
	}
	return false
}

// Count returns the total number of pages in all ranges
func (prs *PageRangeSet) Count() int {
	count := 0
	for _, r := range prs.ranges {
		count += r.End - r.Start + 1
	}
	return count
}

// GetRanges returns all the ranges
func (prs *PageRangeSet) GetRanges() []PageRange {
	return prs.ranges
}

// String returns a string representation of the page ranges
func (prs *PageRangeSet) String() string {
	if len(prs.ranges) == 0 {
		return ""
	}

	var parts []string
	for _, r := range prs.ranges {
		if r.Start == r.End {
			parts = append(parts, fmt.Sprintf("%d", r.Start))
		} else {
			parts = append(parts, fmt.Sprintf("%d-%d", r.Start, r.End))
		}
	}

	return strings.Join(parts, ",")
}

// ValidateAgainstTotal validates that all page numbers are within the total page count
func (prs *PageRangeSet) ValidateAgainstTotal(totalPages int) error {
	for _, r := range prs.ranges {
		if r.Start < 1 {
			return fmt.Errorf("page numbers must be 1 or greater, got: %d", r.Start)
		}
		if r.End > totalPages {
			return fmt.Errorf("page %d exceeds total pages (%d)", r.End, totalPages)
		}
	}
	return nil
}

// GetPageType determines if a page should be treated as an image or text
func GetPageType(pageNum int, imagePageRanges *PageRangeSet) PageType {
	if imagePageRanges != nil && imagePageRanges.Contains(pageNum) {
		return PageTypeImage
	}
	return PageTypeText
}

// PageType represents how a page should be processed
type PageType int

const (
	PageTypeText PageType = iota
	PageTypeImage
)

func (pt PageType) String() string {
	switch pt {
	case PageTypeText:
		return "text"
	case PageTypeImage:
		return "image"
	default:
		return "unknown"
	}
}
