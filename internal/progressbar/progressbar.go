package progressbar

import (
	"math"
	"strings"
)

// Render draws a horizontal progress bar.
// fraction is 0.0 to 1.0, width is the bar width in characters.
func Render(fraction float64, width int) string {
	if fraction < 0 {
		fraction = 0
	}
	if fraction > 1 {
		fraction = 1
	}

	filled := int(math.Round(fraction * float64(width)))
	if filled > width {
		filled = width
	}

	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}
