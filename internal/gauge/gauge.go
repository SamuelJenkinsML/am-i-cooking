package gauge

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	width  = 60
	height = 17
	// Center of the arc
	cx = width / 2
	cy = height - 2
	// Radius (x is scaled 2x for terminal aspect ratio)
	radius = 13.0
)

// Render draws a semicircular gauge at the given percentage (0-100).
// needlePercent is the current animated position of the needle.
func Render(needlePercent float64) string {
	grid := make([][]rune, height)
	colors := make([][]lipgloss.Style, height)
	noStyle := lipgloss.NewStyle()

	for y := range grid {
		grid[y] = make([]rune, width)
		colors[y] = make([]lipgloss.Style, width)
		for x := range grid[y] {
			grid[y][x] = ' '
			colors[y][x] = noStyle
		}
	}

	// Draw arc segments with color gradient
	drawArc(grid, colors)

	// Draw tick marks and labels
	drawTicks(grid, colors)

	// Draw needle
	drawNeedle(grid, colors, needlePercent)

	// Draw center pivot
	if cy >= 0 && cy < height && cx >= 0 && cx < width {
		grid[cy][cx] = '●'
		colors[cy][cx] = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
	}

	// Render to string with colors
	var sb strings.Builder
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			ch := string(grid[y][x])
			styled := colors[y][x].Render(ch)
			sb.WriteString(styled)
		}
		if y < height-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func drawArc(grid [][]rune, colors [][]lipgloss.Style) {
	// Draw semicircle from π (left, 0%) to 0 (right, 100%)
	steps := 200
	for i := 0; i <= steps; i++ {
		// angle goes from π to 0 (left to right)
		angle := math.Pi - (float64(i)/float64(steps))*math.Pi
		pct := float64(i) / float64(steps) * 100.0

		// Outer arc
		x := cx + int(math.Round(radius*2.0*math.Cos(angle)))
		y := cy - int(math.Round(radius*math.Sin(angle)))

		if x >= 0 && x < width && y >= 0 && y < height {
			ch := arcChar(angle)
			grid[y][x] = ch
			colors[y][x] = arcColor(pct)
		}

		// Inner arc (slightly smaller radius)
		innerR := radius - 1.0
		xi := cx + int(math.Round(innerR*2.0*math.Cos(angle)))
		yi := cy - int(math.Round(innerR*math.Sin(angle)))

		if xi >= 0 && xi < width && yi >= 0 && yi < height {
			ch := arcChar(angle)
			grid[yi][xi] = ch
			colors[yi][xi] = arcColor(pct)
		}
	}
}

func drawTicks(grid [][]rune, colors [][]lipgloss.Style) {
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
		// angle: 0% → π, 100% → 0
		angle := math.Pi - (l.pct/100.0)*math.Pi
		outerR := radius + 1.5

		x := cx + int(math.Round(outerR*2.0*math.Cos(angle)))
		y := cy - int(math.Round(outerR*math.Sin(angle)))

		// Place label centered on position
		labelOffset := len(l.label) / 2
		startX := x - labelOffset
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

		for i, ch := range l.label {
			px := startX + i
			if px >= 0 && px < width && y >= 0 && y < height {
				grid[y][px] = ch
				colors[y][px] = style
			}
		}
	}
}

func drawNeedle(grid [][]rune, colors [][]lipgloss.Style, pct float64) {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}

	// angle: 0% → π, 100% → 0
	angle := math.Pi - (pct/100.0)*math.Pi
	needleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true)

	needleLen := radius - 2.0
	steps := 30
	for i := 1; i <= steps; i++ {
		r := (float64(i) / float64(steps)) * needleLen
		x := cx + int(math.Round(r*2.0*math.Cos(angle)))
		y := cy - int(math.Round(r*math.Sin(angle)))

		if x >= 0 && x < width && y >= 0 && y < height {
			ch := needleChar(angle)
			grid[y][x] = ch
			colors[y][x] = needleStyle
		}
	}
}

func arcChar(angle float64) rune {
	// Determine which unicode char best represents the arc at this angle
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

func arcColor(pct float64) lipgloss.Style {
	var color string
	switch {
	case pct < 25:
		color = "#22C55E" // green
	case pct < 50:
		color = "#EAB308" // yellow
	case pct < 75:
		color = "#F97316" // orange
	default:
		color = "#EF4444" // red
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Bold(true)
}

// RenderPercent returns the gauge percentage formatted for display.
func RenderPercent(pct float64) string {
	return fmt.Sprintf("%.0f%%", pct)
}
