package model

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/SamuelJenkinsML/am-i-cooking/internal/calc"
	"github.com/SamuelJenkinsML/am-i-cooking/internal/parser"
	"github.com/SamuelJenkinsML/am-i-cooking/internal/theme"
)

func testTheme() theme.Theme {
	t, _ := theme.ByName("default")
	return t
}

func readyModel() Model {
	m := New("/tmp/test", 5*time.Hour, false, testTheme(), false)
	m.width = 80
	m.height = 40
	m.ready = true
	return m
}

func TestHelpToggle_InitiallyHidden(t *testing.T) {
	m := readyModel()
	if m.showHelp {
		t.Error("expected showHelp to be false initially")
	}
}

func TestHelpToggle_QuestionMarkToggles(t *testing.T) {
	m := readyModel()

	// First press: show help
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = updated.(Model)
	if !m.showHelp {
		t.Error("expected showHelp to be true after first ?")
	}

	// Second press: hide help
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = updated.(Model)
	if m.showHelp {
		t.Error("expected showHelp to be false after second ?")
	}
}

func TestHelpOverlay_RendersInView(t *testing.T) {
	m := readyModel()
	m.showHelp = true

	view := m.View()
	if !strings.Contains(view, "Keybindings") {
		t.Error("expected help overlay to contain 'Keybindings'")
	}
	if !strings.Contains(view, "quit") {
		t.Error("expected help overlay to contain 'quit'")
	}
	if !strings.Contains(view, "?") {
		t.Error("expected help overlay to contain '?'")
	}
}

func TestHelpOverlay_DoesNotRenderWhenHidden(t *testing.T) {
	m := readyModel()
	m.showHelp = false

	view := m.View()
	if strings.Contains(view, "Keybindings") {
		t.Error("expected no help overlay when showHelp is false")
	}
}

func TestHelpOverlay_QuitStillWorks(t *testing.T) {
	m := readyModel()
	m.showHelp = true

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Error("expected quit command when pressing q with help shown")
	}
}

// -- NewSnapshot tests --

func TestNewSnapshot_Ready(t *testing.T) {
	metrics := calc.Metrics{
		CurrentRate:  5000,
		GaugePercent: 45,
		Verdict:      "simmering nicely",
	}
	m := NewSnapshot(metrics, 5*time.Hour, testTheme(), false)

	if !m.ready {
		t.Error("expected snapshot model to be ready")
	}

	view := m.View()
	if strings.Contains(view, "Loading") {
		t.Error("expected snapshot View() to not show Loading")
	}
	if len(view) == 0 {
		t.Error("expected non-empty view from snapshot")
	}
}

func TestNewSnapshot_NeedleMatchesGauge(t *testing.T) {
	metrics := calc.Metrics{GaugePercent: 62.5}
	m := NewSnapshot(metrics, 5*time.Hour, testTheme(), false)

	if m.currentNeedle != 62.5 {
		t.Errorf("expected currentNeedle 62.5, got %f", m.currentNeedle)
	}
}

func TestNewSnapshot_CompactMode(t *testing.T) {
	metrics := calc.Metrics{Verdict: "test"}
	m := NewSnapshot(metrics, 5*time.Hour, testTheme(), true)

	if !m.compact {
		t.Error("expected compact mode when forceCompact is true")
	}

	view := m.View()
	if len(view) == 0 {
		t.Error("expected non-empty compact view")
	}
}

// -- Session breakdown tests --

func TestSessionBreakdown_InitiallyHidden(t *testing.T) {
	m := readyModel()
	if m.showSessionBreakdown {
		t.Error("showSessionBreakdown should be false initially")
	}
}

func TestSessionBreakdown_TabToggles(t *testing.T) {
	m := readyModel()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = result.(Model)
	if !m.showSessionBreakdown {
		t.Error("Tab should toggle showSessionBreakdown to true")
	}

	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = result.(Model)
	if m.showSessionBreakdown {
		t.Error("Tab should toggle showSessionBreakdown back to false")
	}
}

func TestSessionBreakdown_SToggles(t *testing.T) {
	m := readyModel()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = result.(Model)
	if !m.showSessionBreakdown {
		t.Error("'s' should toggle showSessionBreakdown to true")
	}
}

func TestSessionBreakdownView_Renders(t *testing.T) {
	m := readyModel()
	m.showSessionBreakdown = true
	m.sessionMetrics = []calc.SessionMetrics{
		{
			SessionID:           "test-session-123",
			TotalWeightedTokens: 5000,
			TotalRawTokens:      1000,
			Rate:                500,
			EstimatedCost:       0.01,
			PrimaryModel:        "claude-sonnet-4-20250514",
			LastActivity:        time.Now(),
		},
	}

	view := m.View()
	if !strings.Contains(view, "SESSION BREAKDOWN") {
		t.Error("session breakdown view should contain title")
	}
	if !strings.Contains(view, "test-session") {
		t.Error("session breakdown view should contain session ID")
	}
	if !strings.Contains(view, "Tab") {
		t.Error("session breakdown view should contain hint about Tab key")
	}
}

func TestSessionBreakdownView_NoSessions(t *testing.T) {
	m := readyModel()
	m.showSessionBreakdown = true
	m.sessionMetrics = nil

	view := m.View()
	if !strings.Contains(view, "No sessions found") {
		t.Error("should show 'No sessions found' when no sessions")
	}
}

func TestHelpOverlay_IncludesTabBinding(t *testing.T) {
	m := readyModel()
	m.showHelp = true

	view := m.View()
	if !strings.Contains(view, "Tab") {
		t.Error("help overlay should mention Tab key")
	}
	if !strings.Contains(view, "session") {
		t.Error("help overlay should mention session breakdown")
	}
}

func TestRecordsParsedMsg_ComputesSessionMetrics(t *testing.T) {
	m := readyModel()

	msg := RecordsParsedMsg{
		Records: []parser.UsageRecord{
			{
				Timestamp:    time.Now().Add(-2 * time.Minute),
				Model:        "claude-sonnet-4-20250514",
				InputTokens:  100,
				OutputTokens: 50,
				SessionID:    "sess1",
			},
			{
				Timestamp:    time.Now().Add(-1 * time.Minute),
				Model:        "claude-sonnet-4-20250514",
				InputTokens:  200,
				OutputTokens: 100,
				SessionID:    "sess2",
			},
		},
		Offsets: map[string]int64{"test.jsonl": 100},
	}

	result, _ := m.Update(msg)
	m = result.(Model)

	if len(m.sessionMetrics) != 2 {
		t.Fatalf("expected 2 session metrics, got %d", len(m.sessionMetrics))
	}
}

func TestShortModelName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"claude-opus-4-20250514", "Opus"},
		{"claude-sonnet-4-20250514", "Sonnet"},
		{"claude-haiku-4-20250514", "Haiku"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		got := shortModelName(tt.input)
		if got != tt.want {
			t.Errorf("shortModelName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
