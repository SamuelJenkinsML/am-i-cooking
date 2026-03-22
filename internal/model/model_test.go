package model

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/SamuelJenkinsML/am-i-cooking/internal/calc"
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
