package theme

import (
	"fmt"
	"strings"
)

// Theme defines all colors used throughout the UI.
type Theme struct {
	Name string

	// Gauge arc zones (0-25%, 25-50%, 50-75%, 75-100%)
	ArcGreen  string
	ArcYellow string
	ArcOrange string
	ArcRed    string

	// UI chrome
	Title  string
	Border string
	Label  string
	Value  string
	Needle string
	Pivot  string

	// Verdict colors by gauge zone
	VerdictCold   string
	VerdictWarm   string
	VerdictHot    string
	VerdictOnFire string

	// Sparkline
	Sparkline string

	// Progress bar
	ProgressFilled string
	ProgressEmpty  string
}

// ArcColor returns the arc color for a given percentage.
func (t Theme) ArcColor(pct float64) string {
	switch {
	case pct < 25:
		return t.ArcGreen
	case pct < 50:
		return t.ArcYellow
	case pct < 75:
		return t.ArcOrange
	default:
		return t.ArcRed
	}
}

// VerdictColor returns the verdict color for a given gauge percentage.
func (t Theme) VerdictColor(pct float64) string {
	switch {
	case pct < 25:
		return t.VerdictCold
	case pct < 50:
		return t.VerdictWarm
	case pct < 75:
		return t.VerdictHot
	default:
		return t.VerdictOnFire
	}
}

// Default returns the original color scheme.
func Default() Theme {
	return Theme{
		Name:           "default",
		ArcGreen:       "#22C55E",
		ArcYellow:      "#EAB308",
		ArcOrange:      "#F97316",
		ArcRed:         "#EF4444",
		Title:          "#FF6B35",
		Border:         "#444444",
		Label:          "#888888",
		Value:          "#FFFFFF",
		Needle:         "#FFFFFF",
		Pivot:          "#FFFFFF",
		VerdictCold:    "#888888",
		VerdictWarm:    "#EAB308",
		VerdictHot:     "#F97316",
		VerdictOnFire:  "#EF4444",
		Sparkline:      "#FF6B35",
		ProgressFilled: "#22C55E",
		ProgressEmpty:  "#333333",
	}
}

// Minimal returns a subdued, low-contrast theme.
func Minimal() Theme {
	return Theme{
		Name:           "minimal",
		ArcGreen:       "#6B7280",
		ArcYellow:      "#9CA3AF",
		ArcOrange:      "#D1D5DB",
		ArcRed:         "#F3F4F6",
		Title:          "#9CA3AF",
		Border:         "#374151",
		Label:          "#6B7280",
		Value:          "#D1D5DB",
		Needle:         "#D1D5DB",
		Pivot:          "#D1D5DB",
		VerdictCold:    "#6B7280",
		VerdictWarm:    "#9CA3AF",
		VerdictHot:     "#D1D5DB",
		VerdictOnFire:  "#F3F4F6",
		Sparkline:      "#9CA3AF",
		ProgressFilled: "#6B7280",
		ProgressEmpty:  "#374151",
	}
}

// Neon returns a bright cyberpunk-inspired theme.
func Neon() Theme {
	return Theme{
		Name:           "neon",
		ArcGreen:       "#39FF14",
		ArcYellow:      "#FFFF00",
		ArcOrange:      "#FF6EC7",
		ArcRed:         "#FF073A",
		Title:          "#00FFFF",
		Border:         "#6600FF",
		Label:          "#BC13FE",
		Value:          "#39FF14",
		Needle:         "#FFFFFF",
		Pivot:          "#00FFFF",
		VerdictCold:    "#4361EE",
		VerdictWarm:    "#FFFF00",
		VerdictHot:     "#FF6EC7",
		VerdictOnFire:  "#FF073A",
		Sparkline:      "#00FFFF",
		ProgressFilled: "#39FF14",
		ProgressEmpty:  "#1A0033",
	}
}

// Monochrome returns a white/gray only theme.
func Monochrome() Theme {
	return Theme{
		Name:           "monochrome",
		ArcGreen:       "#AAAAAA",
		ArcYellow:      "#CCCCCC",
		ArcOrange:      "#DDDDDD",
		ArcRed:         "#FFFFFF",
		Title:          "#FFFFFF",
		Border:         "#555555",
		Label:          "#777777",
		Value:          "#FFFFFF",
		Needle:         "#FFFFFF",
		Pivot:          "#FFFFFF",
		VerdictCold:    "#777777",
		VerdictWarm:    "#AAAAAA",
		VerdictHot:     "#CCCCCC",
		VerdictOnFire:  "#FFFFFF",
		Sparkline:      "#AAAAAA",
		ProgressFilled: "#CCCCCC",
		ProgressEmpty:  "#333333",
	}
}

// ByName returns a theme by name (case-insensitive).
func ByName(name string) (Theme, error) {
	switch strings.ToLower(name) {
	case "default":
		return Default(), nil
	case "minimal":
		return Minimal(), nil
	case "neon":
		return Neon(), nil
	case "monochrome":
		return Monochrome(), nil
	default:
		return Theme{}, fmt.Errorf("unknown theme %q (available: default, minimal, neon, monochrome)", name)
	}
}
