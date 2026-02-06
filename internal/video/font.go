package video

import (
	"os"
)

// GetBestFontPath returns the path to the best available font for Chinese text.
// Priority: STHeiti -> PingFang -> Arial -> Fallback
func GetBestFontPath() string {
	candidates := []string{
		"msyh.ttf", // Local file has highest priority
		"/Users/leon/Documents/02-code/go/src/github.com/LeonRhapsody/pptTovideo/msyh.ttf",
		"/System/Library/Fonts/STHeiti Medium.ttc",
		"/System/Library/Fonts/STHeiti Light.ttc",
		"/System/Library/Fonts/PingFang.ttc",
		"/Library/Fonts/Microsoft YaHei.ttf", // If user installs it
		"/System/Library/Fonts/Supplemental/Arial.ttf",
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
