package model

import (
	"fmt"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/SamuelJenkinsML/am-i-cooking/internal/calc"
	"github.com/SamuelJenkinsML/am-i-cooking/internal/gauge"
	"github.com/SamuelJenkinsML/am-i-cooking/internal/parser"
	"github.com/SamuelJenkinsML/am-i-cooking/internal/watcher"
)

// AnimTickMsg triggers a gauge animation frame.
type AnimTickMsg struct{}

// RecordsParsedMsg carries newly parsed records.
type RecordsParsedMsg struct {
	Records []parser.UsageRecord
	Offsets map[string]int64 // non-nil only on initial parse
}

// Model is the Bubble Tea model for the dashboard.
type Model struct {
	projectPath string
	windowSize  time.Duration
	allProjects bool

	records []parser.UsageRecord
	offsets map[string]int64
	metrics calc.Metrics

	// Animation state
	currentNeedle float64 // current animated position
	targetNeedle  float64 // target position
	animating     bool

	width  int
	height int
	ready  bool
}

// New creates a new Model.
func New(projectPath string, windowSize time.Duration, allProjects bool) Model {
	return Model{
		projectPath: projectPath,
		windowSize:  windowSize,
		allProjects: allProjects,
		offsets:     make(map[string]int64),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		initialParseCmd(m.projectPath, m.windowSize),
		animTickCmd(),
		watcher.RescanCmd(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case RecordsParsedMsg:
		if msg.Offsets != nil {
			m.offsets = msg.Offsets
		}
		m.records = append(m.records, msg.Records...)
		// Deduplicate and filter to window
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
		m.targetNeedle = m.metrics.GaugePercent
		if !m.animating {
			m.animating = true
		}

	case AnimTickMsg:
		// Ease toward target
		diff := m.targetNeedle - m.currentNeedle
		if math.Abs(diff) < 0.1 {
			m.currentNeedle = m.targetNeedle
			m.animating = false
		} else {
			// Ease-out: move 15% of remaining distance each frame
			m.currentNeedle += diff * 0.15
			m.animating = true
		}
		return m, animTickCmd()

	case watcher.FileChangedMsg:
		// Ensure this file is tracked for incremental reads
		if _, exists := m.offsets[msg.Path]; !exists {
			m.offsets[msg.Path] = 0
		}
		return m, incrementalParseCmd(m.offsets, m.windowSize)

	case watcher.RescanTickMsg:
		// Discover any new JSONL files that appeared since last scan
		parser.DiscoverAndTrack(m.projectPath, m.offsets)
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

	// Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF6B35")).
		Align(lipgloss.Center)

	verdictStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(verdictColor(m.metrics.GaugePercent)).
		Align(lipgloss.Center)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Width(14).
		Align(lipgloss.Right)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#444444")).
		Padding(1, 2)

	// Build gauge
	gaugeStr := gauge.Render(m.currentNeedle)

	// Title
	title := titleStyle.Width(60).Render("AM I COOKING?")

	// Verdict
	verdict := verdictStyle.Width(60).Render(m.metrics.Verdict)

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
			valueStyle.Render(fmt.Sprintf("%s elapsed / %s left",
				formatDuration(m.metrics.WindowElapsed),
				formatDuration(m.metrics.WindowRemaining))),
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
		stats,
	)

	box := borderStyle.Render(content)

	// Center in terminal
	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		box)
}

func verdictColor(pct float64) lipgloss.Color {
	switch {
	case pct < 25:
		return lipgloss.Color("#888888")
	case pct < 50:
		return lipgloss.Color("#EAB308")
	case pct < 75:
		return lipgloss.Color("#F97316")
	default:
		return lipgloss.Color("#EF4444")
	}
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
