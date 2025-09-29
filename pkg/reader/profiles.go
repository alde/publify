package reader

import (
	"fmt"
	"strings"
)

// Available reader profiles
var profiles = map[string]Profile{
	"kobo": {
		Name:         "Kobo Libra Colour",
		Manufacturer: "Kobo",
		Model:        "Libra Colour",
		Capabilities: DeviceCapabilities{
			ScreenWidth:  1264,
			ScreenHeight: 1680,
			DPI:          300,

			SupportsColor: true,
			ColorDepth:    24,

			MaxImageWidth:        1200, // Slightly smaller than screen for margins
			MaxImageHeight:       1600,
			ImageQuality:         85, // Good balance for color display
			CompressionLevel:     "high", // Prioritize file size

			SupportedImageFormats:   []string{"webp", "jpeg", "png"},
			PreferredImageFormat:    "webp", // WebP for best compression

			TargetSizeRatio:         0.25, // Aim for 25% of original PDF size
			StripUnsupportedContent: true,
			AggressiveCompression:   true,
			OptimizeForSize:         true,

			SupportsAdvancedTypography: true,
			DefaultFontSize:            12,
		},
	},
	"kobo-bw": {
		Name:         "Kobo Clara/Libra (B&W)",
		Manufacturer: "Kobo",
		Model:        "Clara/Libra B&W",
		Capabilities: DeviceCapabilities{
			ScreenWidth:  1264,
			ScreenHeight: 1680,
			DPI:          300,

			SupportsColor: false,
			ColorDepth:    8,

			MaxImageWidth:        1200,
			MaxImageHeight:       1600,
			ImageQuality:         90, // Higher quality for grayscale details
			CompressionLevel:     "high",

			SupportedImageFormats:   []string{"webp", "jpeg", "png"},
			PreferredImageFormat:    "webp", // WebP for best compression

			TargetSizeRatio:         0.15, // Very aggressive for B&W - 15% of original
			StripUnsupportedContent: true,
			AggressiveCompression:   true,
			OptimizeForSize:         true,

			SupportsAdvancedTypography: true,
			DefaultFontSize:            12,
		},
	},
	"kindle": {
		Name:         "Kindle Paperwhite",
		Manufacturer: "Amazon",
		Model:        "Paperwhite",
		Capabilities: DeviceCapabilities{
			ScreenWidth:  1236,
			ScreenHeight: 1648,
			DPI:          300,

			SupportsColor: false,
			ColorDepth:    8,

			MaxImageWidth:        1200,
			MaxImageHeight:       1600,
			ImageQuality:         85,
			CompressionLevel:     "high",

			SupportedImageFormats:   []string{"jpeg", "png"}, // Kindle doesn't support WebP
			PreferredImageFormat:    "jpeg",

			TargetSizeRatio:         0.2, // 20% of original - Kindle needs small files
			StripUnsupportedContent: true,
			AggressiveCompression:   true,
			OptimizeForSize:         true,

			SupportsAdvancedTypography: false, // More limited than Kobo
			DefaultFontSize:            12,
		},
	},
	"kindle-oasis": {
		Name:         "Kindle Oasis",
		Manufacturer: "Amazon",
		Model:        "Oasis",
		Capabilities: DeviceCapabilities{
			ScreenWidth:  1264,
			ScreenHeight: 1680,
			DPI:          300,

			SupportsColor: false,
			ColorDepth:    8,

			MaxImageWidth:        1200,
			MaxImageHeight:       1600,
			ImageQuality:         90,
			CompressionLevel:     "high",

			SupportedImageFormats:   []string{"jpeg", "png"}, // Kindle doesn't support WebP
			PreferredImageFormat:    "jpeg",

			TargetSizeRatio:         0.25, // 25% of original
			StripUnsupportedContent: true,
			AggressiveCompression:   true,
			OptimizeForSize:         true,

			SupportsAdvancedTypography: false,
			DefaultFontSize:            12,
		},
	},
	"generic": {
		Name:         "Generic E-Reader",
		Manufacturer: "Generic",
		Model:        "Standard",
		Capabilities: DeviceCapabilities{
			ScreenWidth:  800,
			ScreenHeight: 1200,
			DPI:          200,

			SupportsColor: false,
			ColorDepth:    8,

			MaxImageWidth:        750,
			MaxImageHeight:       1100,
			ImageQuality:         75,
			CompressionLevel:     "high",

			SupportedImageFormats:   []string{"jpeg", "png"}, // Conservative format support
			PreferredImageFormat:    "jpeg",

			TargetSizeRatio:         0.3, // 30% of original - conservative but efficient
			StripUnsupportedContent: true,
			AggressiveCompression:   true,
			OptimizeForSize:         true,

			SupportsAdvancedTypography: false,
			DefaultFontSize:            12,
		},
	},
}

// GetProfile returns a reader profile by name
func GetProfile(name string) (Profile, error) {
	normalizedName := strings.ToLower(strings.TrimSpace(name))

	if profile, exists := profiles[normalizedName]; exists {
		return profile, nil
	}

	// Return available profiles in error
	var available []string
	for key := range profiles {
		available = append(available, key)
	}

	return Profile{}, fmt.Errorf("unknown reader profile '%s'. Available profiles: %v", name, available)
}

// ListProfiles returns all available reader profiles
func ListProfiles() map[string]Profile {
	return profiles
}