package gauge

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/SamuelJenkinsML/am-i-cooking/internal/theme"
)

// Config holds the computed gauge dimensions and theme.
type Config struct {
	Width  int
	Height int
	CX     int
	CY     int
	Radius float64
	Theme  theme.Theme
}

// DefaultConfig returns a config matching the original fixed 60x17 gauge.
func DefaultConfig() Config {
	return NewConfig(80, 24, theme.Default())
}

// NewConfig computes gauge dimensions from available terminal size.
func NewConfig(termWidth, termHeight int, t theme.Theme) Config {
	// Reserve space for border (2), padding (4), stats below (~10 lines), sparkline (1), progress (1)
	availW := termWidth - 6
	availH := termHeight - 14

	// Clamp to reasonable bounds
	if availW > 100 {
		availW = 100
	}
	if availW < 30 {
		availW = 30
	}
	if availH > 25 {
		availH = 25
	}
	if availH < 10 {
		availH = 10
	}

	// Radius limited by height and width
	// Height: radius = availH - 2 (room for labels below arc)
	// Width: need radius*2*2 (diameter * aspect ratio) to fit, so radius = availW/4
	radiusFromH := float64(availH) - 2.0
	radiusFromW := float64(availW) / 4.0
	radius := math.Min(radiusFromH, radiusFromW)
	if radius < 5 {
		radius = 5
	}

	gridH := int(radius) + 4 // extra rows for tick labels + base
	gridW := int(radius*4.0) + 6 // diameter * aspect + label margins

	// Ensure odd width for centering
	if gridW%2 == 0 {
		gridW++
	}

	return Config{
		Width:  gridW,
		Height: gridH,
		CX:     gridW / 2,
		CY:     gridH - 2,
		Radius: radius,
		Theme:  t,
	}
}

// Render draws a semicircular gauge at the given percentage (0-100).
func Render(needlePercent float64, cfg Config) string {
	grid := make([][]rune, cfg.Height)
	colors := make([][]lipgloss.Style, cfg.Height)
	noStyle := lipgloss.NewStyle()

	for y := range grid {
		grid[y] = make([]rune, cfg.Width)
		colors[y] = make([]lipgloss.Style, cfg.Width)
		for x := range grid[y] {
			grid[y][x] = ' '
			colors[y][x] = noStyle
		}
	}

	drawArc(grid, colors, cfg)
	drawTicks(grid, colors, cfg)
	drawNeedle(grid, colors, needlePercent, cfg)

	// Draw center pivot
	if cfg.CY >= 0 && cfg.CY < cfg.Height && cfg.CX >= 0 && cfg.CX < cfg.Width {
		grid[cfg.CY][cfg.CX] = '●'
		colors[cfg.CY][cfg.CX] = lipgloss.NewStyle().Foreground(lipgloss.Color(cfg.Theme.Pivot)).Bold(true)
	}

	var sb strings.Builder
	for y := 0; y < cfg.Height; y++ {
		for x := 0; x < cfg.Width; x++ {
			ch := string(grid[y][x])
			styled := colors[y][x].Render(ch)
			sb.WriteString(styled)
		}
		if y < cfg.Height-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func drawArc(grid [][]rune, colors [][]lipgloss.Style, cfg Config) {
	steps := 200
	for i := 0; i <= steps; i++ {
		angle := math.Pi - (float64(i)/float64(steps))*math.Pi
		pct := float64(i) / float64(steps) * 100.0

		// Outer arc
		x := cfg.CX + int(math.Round(cfg.Radius*2.0*math.Cos(angle)))
		y := cfg.CY - int(math.Round(cfg.Radius*math.Sin(angle)))

		if x >= 0 && x < cfg.Width && y >= 0 && y < cfg.Height {
			grid[y][x] = arcChar(angle)
			colors[y][x] = arcColor(pct, cfg.Theme)
		}

		// Inner arc
		innerR := cfg.Radius - 1.0
		xi := cfg.CX + int(math.Round(innerR*2.0*math.Cos(angle)))
		yi := cfg.CY - int(math.Round(innerR*math.Sin(angle)))

		if xi >= 0 && xi < cfg.Width && yi >= 0 && yi < cfg.Height {
			grid[yi][xi] = arcChar(angle)
			colors[yi][xi] = arcColor(pct, cfg.Theme)
		}
	}
}

func drawTicks(grid [][]rune, colors [][]lipgloss.Style, cfg Config) {
	labels := []struct {
		pct   float64
		label string
	}{
		{0, "0%"},
		{25, "25%"},
		{50, "50%"},
		{75, "75%"},
		{100, "100%"},
	}

	for _, l := range labels {
		angle := math.Pi - (l.pct/100.0)*math.Pi
		outerR := cfg.Radius + 1.5

		x := cfg.CX + int(math.Round(outerR*2.0*math.Cos(angle)))
		y := cfg.CY - int(math.Round(outerR*math.Sin(angle)))

		labelOffset := len(l.label) / 2
		startX := x - labelOffset
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(cfg.Theme.Label))

		for i, ch := range l.label {
			px := startX + i
			if px >= 0 && px < cfg.Width && y >= 0 && y < cfg.Height {
				grid[y][px] = ch
				colors[y][px] = style
			}
		}
	}
}

func drawNeedle(grid [][]rune, colors [][]lipgloss.Style, pct float64, cfg Config) {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}

	angle := math.Pi - (pct/100.0)*math.Pi
	needleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(cfg.Theme.Needle)).Bold(true)

	needleLen := cfg.Radius - 2.0
	steps := 30
	for i := 1; i <= steps; i++ {
		r := (float64(i) / float64(steps)) * needleLen
		x := cfg.CX + int(math.Round(r*2.0*math.Cos(angle)))
		y := cfg.CY - int(math.Round(r*math.Sin(angle)))

		if x >= 0 && x < cfg.Width && y >= 0 && y < cfg.Height {
			grid[y][x] = needleChar(angle)
			colors[y][x] = needleStyle
		}
	}
}

func arcChar(angle float64) rune {
	deg := angle * 180.0 / math.Pi
	switch {
	case deg > 160 || deg < 20:
		return '━'
	case deg > 70 && deg < 110:
		return '┃'
	case deg >= 110 && deg <= 160:
		return '╲'
	default:
		return '╱'
	}
}

func needleChar(angle float64) rune {
	deg := angle * 180.0 / math.Pi
	switch {
	case deg > 160 || deg < 20:
		return '─'
	case deg > 70 && deg < 110:
		return '│'
	case deg >= 110 && deg <= 160:
		return '╲'
	default:
		return '╱'
	}
}

func arcColor(pct float64, t theme.Theme) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(t.ArcColor(pct))).Bold(true)
}

// RenderPercent returns the gauge percentage formatted for display.
func RenderPercent(pct float64) string {
	return fmt.Sprintf("%.0f%%", pct)
}
