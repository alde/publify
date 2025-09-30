package converter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/alde/publify/pkg/reader"
)

// EPUBOptimizer handles EPUB content optimization for specific readers
type EPUBOptimizer struct {
	profile reader.Profile
}

// NewEPUBOptimizer creates a new EPUB optimizer
func NewEPUBOptimizer(profile reader.Profile) *EPUBOptimizer {
	return &EPUBOptimizer{
		profile: profile,
	}
}

// OptimizeHTML optimizes HTML content for the target reader
func (eo *EPUBOptimizer) OptimizeHTML(html string) string {
	if !eo.profile.Capabilities.StripUnsupportedContent {
		return html
	}

	// Start with the original HTML
	optimized := html

	// Remove unnecessary whitespace
	optimized = eo.minifyHTML(optimized)

	// Strip unsupported CSS properties
	optimized = eo.stripUnsupportedCSS(optimized)

	// Remove advanced typography if not supported
	if !eo.profile.Capabilities.SupportsAdvancedTypography {
		optimized = eo.stripAdvancedTypography(optimized)
	}

	// Remove color information for grayscale readers
	if !eo.profile.Capabilities.SupportsColor {
		optimized = eo.stripColorInformation(optimized)
	}

	// Optimize font sizing
	optimized = eo.optimizeFonts(optimized)

	return optimized
}

// minifyHTML removes unnecessary whitespace and formatting
func (eo *EPUBOptimizer) minifyHTML(html string) string {
	// Remove comments
	html = regexp.MustCompile(`<!--.*?-->`).ReplaceAllString(html, "")

	// Normalize whitespace between tags
	html = regexp.MustCompile(`>\s+<`).ReplaceAllString(html, "><")

	// Remove excessive whitespace within text content
	html = regexp.MustCompile(`\s{2,}`).ReplaceAllString(html, " ")

	// Remove trailing whitespace from lines
	lines := strings.Split(html, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	html = strings.Join(lines, "\n")

	// Remove empty lines
	html = regexp.MustCompile(`\n\s*\n`).ReplaceAllString(html, "\n")

	return strings.TrimSpace(html)
}

// stripUnsupportedCSS removes CSS properties that the reader doesn't support
func (eo *EPUBOptimizer) stripUnsupportedCSS(html string) string {
	// Remove advanced CSS properties that many e-readers don't support
	unsupportedProperties := []string{
		`box-shadow[^;]*;`,
		`text-shadow[^;]*;`,
		`border-radius[^;]*;`,
		`gradient[^;]*;`,
		`transform[^;]*;`,
		`animation[^;]*;`,
		`transition[^;]*;`,
		`filter[^;]*;`,
		`backdrop-filter[^;]*;`,
		`clip-path[^;]*;`,
		`mask[^;]*;`,
	}

	for _, prop := range unsupportedProperties {
		html = regexp.MustCompile(prop).ReplaceAllString(html, "")
	}

	// Clean up empty style attributes
	html = regexp.MustCompile(`style="[\s]*"`).ReplaceAllString(html, "")

	return html
}

// stripAdvancedTypography removes advanced typography features
func (eo *EPUBOptimizer) stripAdvancedTypography(html string) string {
	// Remove font features that basic readers don't support
	advancedFeatures := []string{
		`font-feature-settings[^;]*;`,
		`font-variant-ligatures[^;]*;`,
		`font-variant-caps[^;]*;`,
		`font-variant-numeric[^;]*;`,
		`font-kerning[^;]*;`,
		`text-rendering[^;]*;`,
		`-webkit-font-smoothing[^;]*;`,
		`-moz-osx-font-smoothing[^;]*;`,
	}

	for _, feature := range advancedFeatures {
		html = regexp.MustCompile(feature).ReplaceAllString(html, "")
	}

	return html
}

// stripColorInformation removes color-related CSS for grayscale readers
func (eo *EPUBOptimizer) stripColorInformation(html string) string {
	// Remove color properties
	colorProperties := []string{
		`color:\s*[^;]*;`,
		`background-color:\s*[^;]*;`,
		`border-color:\s*[^;]*;`,
		`outline-color:\s*[^;]*;`,
		`text-decoration-color:\s*[^;]*;`,
	}

	for _, prop := range colorProperties {
		html = regexp.MustCompile(prop).ReplaceAllString(html, "")
	}

	// Replace color values in compound properties
	html = regexp.MustCompile(`#[0-9a-fA-F]{3,6}`).ReplaceAllString(html, "")
	html = regexp.MustCompile(`rgb\([^)]*\)`).ReplaceAllString(html, "")
	html = regexp.MustCompile(`rgba\([^)]*\)`).ReplaceAllString(html, "")
	html = regexp.MustCompile(`hsl\([^)]*\)`).ReplaceAllString(html, "")
	html = regexp.MustCompile(`hsla\([^)]*\)`).ReplaceAllString(html, "")

	return html
}

// optimizeFonts adjusts font settings for the target reader
func (eo *EPUBOptimizer) optimizeFonts(html string) string {
	// Simplify font stacks to basic fonts that e-readers support
	basicFontStack := "serif"
	if eo.profile.Manufacturer == "Amazon" {
		basicFontStack = "serif" // Kindle prefers serif
	}

	// Replace complex font families with basic ones
	html = regexp.MustCompile(`font-family:\s*[^;]*;`).ReplaceAllString(html, fmt.Sprintf("font-family: %s;", basicFontStack))

	// Set reasonable default font size
	defaultSize := eo.profile.Capabilities.DefaultFontSize
	html = regexp.MustCompile(`font-size:\s*[^;]*;`).ReplaceAllString(html, fmt.Sprintf("font-size: %dpt;", defaultSize))

	return html
}

// OptimizeCSS optimizes standalone CSS content
func (eo *EPUBOptimizer) OptimizeCSS(css string) string {
	if !eo.profile.Capabilities.StripUnsupportedContent {
		return css
	}

	// Remove comments
	css = regexp.MustCompile(`/\*.*?\*/`).ReplaceAllString(css, "")

	// Minify CSS
	css = eo.minifyCSS(css)

	// Remove unsupported properties
	css = eo.stripUnsupportedCSSProperties(css)

	// Optimize for grayscale if needed
	if !eo.profile.Capabilities.SupportsColor {
		css = eo.stripCSSColors(css)
	}

	return css
}

// minifyCSS removes unnecessary whitespace from CSS
func (eo *EPUBOptimizer) minifyCSS(css string) string {
	// Remove extra whitespace
	css = regexp.MustCompile(`\s+`).ReplaceAllString(css, " ")

	// Remove spaces around certain characters
	css = regexp.MustCompile(`\s*{\s*`).ReplaceAllString(css, "{")
	css = regexp.MustCompile(`\s*}\s*`).ReplaceAllString(css, "}")
	css = regexp.MustCompile(`\s*:\s*`).ReplaceAllString(css, ":")
	css = regexp.MustCompile(`\s*;\s*`).ReplaceAllString(css, ";")
	css = regexp.MustCompile(`\s*,\s*`).ReplaceAllString(css, ",")

	// Remove trailing semicolons before closing braces
	css = regexp.MustCompile(`;}`).ReplaceAllString(css, "}")

	return strings.TrimSpace(css)
}

// stripUnsupportedCSSProperties removes CSS properties not supported by e-readers
func (eo *EPUBOptimizer) stripUnsupportedCSSProperties(css string) string {
	// List of commonly unsupported CSS properties in e-readers
	unsupported := []string{
		`[^}]*box-shadow[^;]*;`,
		`[^}]*text-shadow[^;]*;`,
		`[^}]*border-radius[^;]*;`,
		`[^}]*gradient[^;]*;`,
		`[^}]*transform[^;]*;`,
		`[^}]*animation[^;]*;`,
		`[^}]*transition[^;]*;`,
		`[^}]*filter[^;]*;`,
		`[^}]*@keyframes[^}]*}`,
		`[^}]*@media[^}]*}`,
	}

	for _, pattern := range unsupported {
		css = regexp.MustCompile(pattern).ReplaceAllString(css, "")
	}

	return css
}

// stripCSSColors removes color-related CSS properties
func (eo *EPUBOptimizer) stripCSSColors(css string) string {
	// Remove color properties
	colorProps := []string{
		`[^}]*color[^;]*;`,
		`[^}]*background-color[^;]*;`,
		`[^}]*border-color[^;]*;`,
	}

	for _, prop := range colorProps {
		css = regexp.MustCompile(prop).ReplaceAllString(css, "")
	}

	return css
}

// OptimizeText optimizes text content for file size
func (eo *EPUBOptimizer) OptimizeText(text string) string {
	if !eo.profile.Capabilities.AggressiveCompression {
		return text
	}

	// Remove excessive whitespace
	text = regexp.MustCompile(`[ \t]+`).ReplaceAllString(text, " ")

	// Normalize line endings
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// Remove excessive line breaks (more than 2)
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")

	// Trim whitespace from lines
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}

	return strings.Join(lines, "\n")
}

// CalculateOptimizationStats returns statistics about optimization
func (eo *EPUBOptimizer) CalculateOptimizationStats(original, optimized string) OptimizationStats {
	originalSize := len([]byte(original))
	optimizedSize := len([]byte(optimized))

	reduction := 0.0
	if originalSize > 0 {
		reduction = float64(originalSize-optimizedSize) / float64(originalSize) * 100
	}

	return OptimizationStats{
		OriginalSize:  originalSize,
		OptimizedSize: optimizedSize,
		SizeReduction: reduction,
		BytesSaved:    originalSize - optimizedSize,
	}
}

// OptimizationStats contains optimization metrics
type OptimizationStats struct {
	OriginalSize  int
	OptimizedSize int
	SizeReduction float64 // Percentage
	BytesSaved    int
}
