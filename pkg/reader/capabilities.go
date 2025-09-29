package reader

// DeviceCapabilities defines the technical capabilities of an e-reader
type DeviceCapabilities struct {
	// Display specifications
	ScreenWidth  int // Width in pixels
	ScreenHeight int // Height in pixels
	DPI          int // Dots per inch

	// Color and format support
	SupportsColor bool
	ColorDepth    int // Bits per pixel (1 for grayscale, 8 for 256 colors, 24 for full color)

	// Image processing preferences
	MaxImageWidth  int     // Maximum recommended image width in pixels
	MaxImageHeight int     // Maximum recommended image height in pixels
	ImageQuality   int     // JPEG quality (1-100, higher = better quality)
	CompressionLevel string // "low", "medium", "high" - affects file size vs quality

	// Format preferences
	SupportedImageFormats []string // Supported formats in order of preference: ["webp", "jpeg", "png"]
	PreferredImageFormat  string   // Primary format to use

	// Size optimization settings
	TargetSizeRatio       float64 // Target output size as ratio of input (e.g., 0.3 = 30% of original)
	StripUnsupportedContent bool   // Remove content the reader can't use
	AggressiveCompression   bool   // Use maximum compression for file size
	OptimizeForSize       bool     // Prioritize file size over quality

	// Text rendering
	SupportsAdvancedTypography bool // Ligatures, kerning, etc.
	DefaultFontSize            int  // Recommended base font size in points
}

// Profile represents a complete e-reader profile
type Profile struct {
	Name         string
	Manufacturer string
	Model        string
	Capabilities DeviceCapabilities
}

// ImageProcessingSettings returns optimized image settings for this profile
func (p *Profile) ImageProcessingSettings() ImageSettings {
	return ImageSettings{
		MaxWidth:         p.Capabilities.MaxImageWidth,
		MaxHeight:        p.Capabilities.MaxImageHeight,
		Quality:          p.Capabilities.ImageQuality,
		Format:           p.Capabilities.PreferredImageFormat,
		Grayscale:        !p.Capabilities.SupportsColor,
		CompressionLevel: p.Capabilities.CompressionLevel,
	}
}

// ImageSettings contains image processing parameters
type ImageSettings struct {
	MaxWidth         int
	MaxHeight        int
	Quality          int    // JPEG quality 1-100
	Format           string // "jpeg", "png", "auto"
	Grayscale        bool
	CompressionLevel string
}