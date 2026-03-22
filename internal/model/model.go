package model

import (
	"fmt"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"

	"github.com/SamuelJenkinsML/am-i-cooking/internal/calc"
	"github.com/SamuelJenkinsML/am-i-cooking/internal/gauge"
	"github.com/SamuelJenkinsML/am-i-cooking/internal/parser"
	"github.com/SamuelJenkinsML/am-i-cooking/internal/sparkline"
	"github.com/SamuelJenkinsML/am-i-cooking/internal/theme"
	"github.com/SamuelJenkinsML/am-i-cooking/internal/watcher"
)

const compactThreshold = 20

// AnimTickMsg triggers a gauge animation frame.
type AnimTickMsg struct{}

// SparklineSampleTickMsg triggers a sparkline sample capture.
type SparklineSampleTickMsg struct{}

// RecordsParsedMsg carries newly parsed records.
type RecordsParsedMsg struct {
	Records []parser.UsageRecord
	Offsets map[string]int64
}

// Model is the Bubble Tea model for the dashboard.
type Model struct {
	projectPath string
	windowSize  time.Duration
	allProjects bool
	theme       theme.Theme

	records []parser.UsageRecord
	offsets map[string]int64
	metrics calc.Metrics

	// Animation state
	currentNeedle float64
	targetNeedle  float64
	animating     bool

	// Verdict animation
	prevVerdict      string
	prevGaugePercent float64
	verdictTransition float64
	verdictAnimating  bool

	// Sparkline
	sparkBuf *sparkline.Buffer

	// Compact mode
	compact      bool
	forceCompact bool

	// Help overlay
	showHelp bool

	// Session breakdown
	showSessionBreakdown bool
	sessionMetrics       []calc.SessionMetrics

	width  int
	height int
	ready  bool
}

// New creates a new Model.
func New(projectPath string, windowSize time.Duration, allProjects bool, t theme.Theme, forceCompact bool) Model {
	return Model{
		projectPath:  projectPath,
		windowSize:   windowSize,
		allProjects:  allProjects,
		theme:        t,
		forceCompact: forceCompact,
		offsets:      make(map[string]int64),
		sparkBuf:     sparkline.NewBuffer(60),
	}
}

// NewSnapshot creates a ready-to-render Model with pre-computed metrics.
// Used by --once for single-frame output without the Bubble Tea event loop.
func NewSnapshot(metrics calc.Metrics, windowSize time.Duration, t theme.Theme, forceCompact bool) Model {
	w, h := 80, 40
	if forceCompact {
		h = 10
	}
	return Model{
		windowSize:    windowSize,
		theme:         t,
		forceCompact:  forceCompact,
		compact:       forceCompact,
		metrics:       metrics,
		currentNeedle: metrics.GaugePercent,
		targetNeedle:  metrics.GaugePercent,
		prevVerdict:   metrics.Verdict,
		width:         w,
		height:        h,
		ready:         true,
		offsets:       make(map[string]int64),
		sparkBuf:      sparkline.NewBuffer(60),
	}
}

func (m Model) Init() tea.Cmd {
	var parseCmd tea.Cmd
	if m.allProjects {
		parseCmd = initialParseAllCmd(m.windowSize)
	} else {
		parseCmd = initialParseCmd(m.projectPath, m.windowSize)
	}
	return tea.Batch(
		parseCmd,
		animTickCmd(),
		sparklineSampleCmd(),
		watcher.RescanCmd(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		case "tab", "s":
			m.showSessionBreakdown = !m.showSessionBreakdown
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.compact = m.forceCompact || msg.Height < compactThreshold
		m.ready = true

	case RecordsParsedMsg:
		if msg.Offsets != nil {
			m.offsets = msg.Offsets
		}
		m.records = append(m.records, msg.Records...)
		cutoff := time.Now().Add(-m.windowSize)
		seen := make(map[string]bool)
		var filtered []parser.UsageRecord
		for _, r := range m.records {
			if r.Timestamp.Before(cutoff) {
				continue
			}
			key := r.SessionID + r.Timestamp.String() + r.Model
			if seen[key] {
				continue
			}
			seen[key] = true
			filtered = append(filtered, r)
		}
		m.records = filtered
		m.metrics = calc.Calculate(m.records, m.windowSize)
		m.sessionMetrics = calc.CalculateBySession(m.records, m.windowSize)
		m.targetNeedle = m.metrics.GaugePercent
		if !m.animating {
			m.animating = true
		}

		// Verdict animation: detect change
		if m.metrics.Verdict != m.prevVerdict && m.prevVerdict != "" {
			m.prevGaugePercent = m.currentNeedle // use current needle as "old" percent
			m.verdictTransition = 0.0
			m.verdictAnimating = true
		}
		m.prevVerdict = m.metrics.Verdict

	case AnimTickMsg:
		// Needle ease-out
		diff := m.targetNeedle - m.currentNeedle
		if math.Abs(diff) < 0.1 {
			m.currentNeedle = m.targetNeedle
			m.animating = false
		} else {
			m.currentNeedle += diff * 0.15
			m.animating = true
		}

		// Verdict color transition
		if m.verdictAnimating {
			m.verdictTransition += 0.1
			if m.verdictTransition >= 1.0 {
				m.verdictTransition = 1.0
				m.verdictAnimating = false
			}
		}

		return m, animTickCmd()

	case SparklineSampleTickMsg:
		m.sparkBuf.Add(m.metrics.CurrentRate)
		return m, sparklineSampleCmd()

	case watcher.FileChangedMsg:
		if _, exists := m.offsets[msg.Path]; !exists {
			m.offsets[msg.Path] = 0
		}
		return m, incrementalParseCmd(m.offsets, m.windowSize)

	case watcher.RescanTickMsg:
		if m.allProjects {
			parser.DiscoverAndTrackAll("", m.offsets)
		} else {
			parser.DiscoverAndTrack(m.projectPath, m.offsets)
		}
		return m, tea.Batch(
			incrementalParseCmd(m.offsets, m.windowSize),
			watcher.RescanCmd(),
		)
	}

	return m, nil
}

func (m Model) View() string {
	if !m.ready {
		return "\n  Loading..."
	}

	if m.showHelp {
		return m.helpView()
	}

	if m.showSessionBreakdown {
		return m.sessionBreakdownView()
	}

	if m.compact {
		return m.compactView()
	}

	return m.fullView()
}

var helpBindings = []struct {
	Key  string
	Desc string
}{
	{"?", "toggle this help"},
	{"Tab / s", "session breakdown"},
	{"q", "quit"},
	{"ctrl+c", "quit"},
}

func (m Model) helpView() string {
	t := m.theme

	lines := []string{"  Keybindings", "  " + strings.Repeat("─", 24)}
	for _, b := range helpBindings {
		lines = append(lines, fmt.Sprintf("  %-12s %s", b.Key, b.Desc))
	}

	helpContent := strings.Join(lines, "\n")

	helpBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Border)).
		Padding(1, 2).
		Render(helpContent)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, helpBox)
}

func (m Model) fullView() string {
	t := m.theme

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(t.Title)).
		Align(lipgloss.Center)

	verdictStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.currentVerdictColor()).
		Align(lipgloss.Center)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Label)).
		Width(14).
		Align(lipgloss.Right)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Value)).
		Bold(true)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Border)).
		Padding(1, 2)

	// Compute responsive gauge config
	cfg := gauge.NewConfig(m.width, m.height, t)
	gaugeStr := gauge.Render(m.currentNeedle, cfg)

	contentWidth := cfg.Width
	if contentWidth < 40 {
		contentWidth = 40
	}

	title := titleStyle.Width(contentWidth).Render("AM I COOKING?")
	verdict := verdictStyle.Width(contentWidth).Render(m.metrics.Verdict)

	// Sparkline
	sparkWidth := contentWidth - 4
	if sparkWidth < 10 {
		sparkWidth = 10
	}
	sparkStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Sparkline))
	sparkStr := sparkStyle.Render(sparkline.Render(m.sparkBuf.Samples(), sparkWidth))
	sparkLine := lipgloss.NewStyle().Width(contentWidth).Align(lipgloss.Center).Render(sparkStr)

	// Progress bar for window elapsed
	windowFraction := 0.0
	if m.metrics.WindowSize > 0 {
		windowFraction = m.metrics.WindowElapsed.Seconds() / m.metrics.WindowSize.Seconds()
	}
	barStyled := renderStyledBar(windowFraction, 20, t.ProgressFilled, t.ProgressEmpty)

	// Stats
	stats := strings.Join([]string{
		lipgloss.JoinHorizontal(lipgloss.Top,
			labelStyle.Render("Burn Rate"),
			"  ",
			valueStyle.Render(formatRate(m.metrics.CurrentRate)),
		),
		lipgloss.JoinHorizontal(lipgloss.Top,
			labelStyle.Render("Sustained"),
			"  ",
			valueStyle.Render(formatRate(m.metrics.SustainedRate)),
		),
		lipgloss.JoinHorizontal(lipgloss.Top,
			labelStyle.Render("Window"),
			"  ",
			valueStyle.Render(fmt.Sprintf("%s / %s  ",
				formatDuration(m.metrics.WindowElapsed),
				formatDuration(m.metrics.WindowSize))),
			barStyled,
		),
		lipgloss.JoinHorizontal(lipgloss.Top,
			labelStyle.Render("Total"),
			"  ",
			valueStyle.Render(formatTokens(m.metrics.TotalRawTokens)),
		),
		lipgloss.JoinHorizontal(lipgloss.Top,
			labelStyle.Render("Cost"),
			"  ",
			valueStyle.Render(fmt.Sprintf("~£%.2f this window", m.metrics.EstimatedCost)),
		),
		lipgloss.JoinHorizontal(lipgloss.Top,
			labelStyle.Render("Models"),
			"  ",
			valueStyle.Render(formatModels(m.metrics)),
		),
	}, "\n")

	content := lipgloss.JoinVertical(lipgloss.Center,
		title,
		"",
		gaugeStr,
		"",
		verdict,
		"",
		sparkLine,
		"",
		stats,
	)

	box := borderStyle.Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m Model) compactView() string {
	t := m.theme

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(t.Title))

	verdictStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.currentVerdictColor())

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Label))

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Value)).
		Bold(true)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Border)).
		Padding(0, 1)

	// Mini gauge bar
	gaugeFraction := m.currentNeedle / 100.0
	barStyled := renderStyledBar(gaugeFraction, 12, t.ArcColor(m.currentNeedle), t.ProgressEmpty)

	line1 := lipgloss.JoinHorizontal(lipgloss.Top,
		titleStyle.Render("AM I COOKING?"),
		"  ",
		barStyled,
		"  ",
		valueStyle.Render(fmt.Sprintf("%.0f%%", m.currentNeedle)),
		"  ",
		verdictStyle.Render(m.metrics.Verdict),
	)

	line2 := lipgloss.JoinHorizontal(lipgloss.Top,
		labelStyle.Render("Rate: "),
		valueStyle.Render(formatRate(m.metrics.CurrentRate)),
		labelStyle.Render(" | Sustained: "),
		valueStyle.Render(formatRate(m.metrics.SustainedRate)),
		labelStyle.Render(" | Cost: "),
		valueStyle.Render(fmt.Sprintf("~£%.2f", m.metrics.EstimatedCost)),
	)

	windowFraction := 0.0
	if m.metrics.WindowSize > 0 {
		windowFraction = m.metrics.WindowElapsed.Seconds() / m.metrics.WindowSize.Seconds()
	}
	windowBar := renderStyledBar(windowFraction, 12, t.ProgressFilled, t.ProgressEmpty)

	line3 := lipgloss.JoinHorizontal(lipgloss.Top,
		labelStyle.Render("Window: "),
		valueStyle.Render(fmt.Sprintf("%s / %s", formatDuration(m.metrics.WindowElapsed), formatDuration(m.metrics.WindowSize))),
		"  ",
		windowBar,
		labelStyle.Render(" | Models: "),
		valueStyle.Render(formatModels(m.metrics)),
	)

	content := strings.Join([]string{line1, line2, line3}, "\n")
	box := borderStyle.Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m Model) currentVerdictColor() lipgloss.Color {
	t := m.theme
	newColor := t.VerdictColor(m.metrics.GaugePercent)

	if !m.verdictAnimating {
		return lipgloss.Color(newColor)
	}

	oldColor := t.VerdictColor(m.prevGaugePercent)
	blended := blendColors(oldColor, newColor, m.verdictTransition)
	return lipgloss.Color(blended)
}

func renderStyledBar(fraction float64, width int, filledColor, emptyColor string) string {
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
	filledStr := lipgloss.NewStyle().Foreground(lipgloss.Color(filledColor)).Render(strings.Repeat("█", filled))
	emptyStr := lipgloss.NewStyle().Foreground(lipgloss.Color(emptyColor)).Render(strings.Repeat("░", width-filled))
	return filledStr + emptyStr
}

func blendColors(from, to string, t float64) string {
	c1, err1 := colorful.Hex(from)
	c2, err2 := colorful.Hex(to)
	if err1 != nil || err2 != nil {
		return to
	}
	blended := c1.BlendLab(c2, t)
	return blended.Hex()
}

func formatRate(tokPerMin float64) string {
	if tokPerMin < 1 {
		return "0 tok/min"
	}
	return fmt.Sprintf("%s tok/min", formatNumber(int(tokPerMin)))
}

func formatTokens(n int) string {
	return fmt.Sprintf("%s tokens", formatNumber(n))
}

func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1_000_000 {
		return fmt.Sprintf("%d,%03d", n/1000, n%1000)
	}
	return fmt.Sprintf("%d,%03d,%03d", n/1_000_000, (n%1_000_000)/1000, n%1000)
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func formatModels(m calc.Metrics) string {
	var parts []string
	if m.OpusPercent > 0 {
		parts = append(parts, fmt.Sprintf("Opus: %.0f%%", m.OpusPercent))
	}
	if m.SonnetPercent > 0 {
		parts = append(parts, fmt.Sprintf("Sonnet: %.0f%%", m.SonnetPercent))
	}
	if m.HaikuPercent > 0 {
		parts = append(parts, fmt.Sprintf("Haiku: %.0f%%", m.HaikuPercent))
	}
	if len(parts) == 0 {
		return "—"
	}
	return strings.Join(parts, " | ")
}

// Commands

func animTickCmd() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
		return AnimTickMsg{}
	})
}

func sparklineSampleCmd() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return SparklineSampleTickMsg{}
	})
}

func initialParseCmd(projectPath string, window time.Duration) tea.Cmd {
	return func() tea.Msg {
		records, offsets, err := parser.ParseAll(projectPath, window)
		if err != nil {
			return RecordsParsedMsg{}
		}
		return RecordsParsedMsg{Records: records, Offsets: offsets}
	}
}

func incrementalParseCmd(offsets map[string]int64, window time.Duration) tea.Cmd {
	return func() tea.Msg {
		records, err := parser.ParseIncremental(offsets, window)
		if err != nil {
			return RecordsParsedMsg{}
		}
		return RecordsParsedMsg{Records: records}
	}
}

func initialParseAllCmd(window time.Duration) tea.Cmd {
	return func() tea.Msg {
		records, offsets, err := parser.ParseAllProjects("", window)
		if err != nil {
			return RecordsParsedMsg{}
		}
		return RecordsParsedMsg{Records: records, Offsets: offsets}
	}
}

func (m Model) sessionBreakdownView() string {
	t := m.theme

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(t.Title)).
		Align(lipgloss.Center)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(t.Label))

	rowStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Value))

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Border)).
		Padding(1, 2)

	contentWidth := m.width - 10
	if contentWidth < 60 {
		contentWidth = 60
	}

	title := titleStyle.Width(contentWidth).Render("SESSION BREAKDOWN")

	header := headerStyle.Render(fmt.Sprintf("  %-14s  %12s  %14s  %8s  %-12s",
		"Session", "Tokens", "Rate", "Cost", "Model"))

	separator := strings.Repeat("─", contentWidth)

	maxRows := m.height - 10
	if maxRows < 5 {
		maxRows = 5
	}

	var rows []string
	for i, sm := range m.sessionMetrics {
		if i >= maxRows {
			break
		}
		sid := sm.SessionID
		if len(sid) > 12 {
			sid = sid[:12] + ".."
		}
		rows = append(rows, rowStyle.Render(fmt.Sprintf("  %-14s  %12s  %14s  £%6.2f  %-12s",
			sid,
			formatTokens(sm.TotalRawTokens),
			formatRate(sm.Rate),
			sm.EstimatedCost,
			shortModelName(sm.PrimaryModel),
		)))
	}

	if len(rows) == 0 {
		rows = append(rows, rowStyle.Render("  No sessions found"))
	}

	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Label)).
		Render("  Press Tab or s to return")

	content := lipgloss.JoinVertical(lipgloss.Left,
		title, "", header, separator,
		strings.Join(rows, "\n"), "", hint,
	)

	box := borderStyle.Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func shortModelName(model string) string {
	m := strings.ToLower(model)
	switch {
	case strings.Contains(m, "opus"):
		return "Opus"
	case strings.Contains(m, "haiku"):
		return "Haiku"
	case strings.Contains(m, "sonnet"):
		return "Sonnet"
	default:
		if len(model) > 12 {
			return model[:12]
		}
		return model
	}
}
