package gauge

import (
	"strings"
	"testing"

	"github.com/SamuelJenkinsML/am-i-cooking/internal/theme"
)

func defaultCfg() Config {
	return NewConfig(80, 24, theme.Default())
}

func TestRenderNotEmpty(t *testing.T) {
	result := Render(50, defaultCfg())
	if len(result) == 0 {
		t.Error("Expected non-empty gauge render")
	}
}

func TestRenderContainsArcChars(t *testing.T) {
	result := Render(50, defaultCfg())
	if !strings.ContainsAny(result, "━┃╱╲") {
		t.Error("Expected arc characters in gauge render")
	}
}

func TestRenderContainsPivot(t *testing.T) {
	result := Render(50, defaultCfg())
	if !strings.Contains(result, "●") {
		t.Error("Expected pivot character in gauge render")
	}
}

func TestRenderContainsNeedle(t *testing.T) {
	result := Render(75, defaultCfg())
	if !strings.ContainsAny(result, "─│╲╱") {
		t.Error("Expected needle characters in gauge render")
	}
}

func TestRenderZeroPercent(t *testing.T) {
	result := Render(0, defaultCfg())
	if len(result) == 0 {
		t.Error("Expected non-empty gauge at 0%")
	}
}

func TestRenderHundredPercent(t *testing.T) {
	result := Render(100, defaultCfg())
	if len(result) == 0 {
		t.Error("Expected non-empty gauge at 100%")
	}
}

func TestRenderPercentFormat(t *testing.T) {
	got := RenderPercent(42.7)
	if got != "43%" {
		t.Errorf("RenderPercent(42.7) = %q, want 43%%", got)
	}
}

func TestNewConfigStandardTerminal(t *testing.T) {
	cfg := NewConfig(80, 24, theme.Default())
	if cfg.Width <= 0 || cfg.Height <= 0 {
		t.Errorf("Config has invalid dimensions: %dx%d", cfg.Width, cfg.Height)
	}
	if cfg.Radius < 5 {
		t.Errorf("Config radius %f too small", cfg.Radius)
	}
	if cfg.CX != cfg.Width/2 {
		t.Errorf("CX %d should be Width/2 (%d)", cfg.CX, cfg.Width/2)
	}
}

func TestNewConfigWideTerminal(t *testing.T) {
	cfg := NewConfig(200, 50, theme.Default())
	// Should clamp available dimensions
	if cfg.Radius > 25 {
		t.Errorf("Config radius %f should be clamped for wide terminal", cfg.Radius)
	}
}

func TestNewConfigNarrowTerminal(t *testing.T) {
	cfg := NewConfig(40, 15, theme.Default())
	if cfg.Radius < 5 {
		t.Errorf("Config radius %f should be at least 5", cfg.Radius)
	}
}

func TestRenderResponsiveLineCount(t *testing.T) {
	sizes := []struct {
		w, h int
	}{
		{80, 24},
		{120, 40},
		{60, 18},
	}
	for _, s := range sizes {
		cfg := NewConfig(s.w, s.h, theme.Default())
		result := Render(50, cfg)
		lines := strings.Split(result, "\n")
		if len(lines) != cfg.Height {
			t.Errorf("Terminal %dx%d: got %d lines, want %d", s.w, s.h, len(lines), cfg.Height)
		}
	}
}

func TestRenderWithDifferentThemes(t *testing.T) {
	themes := []string{"default", "minimal", "neon", "monochrome"}
	for _, name := range themes {
		th, _ := theme.ByName(name)
		cfg := NewConfig(80, 24, th)
		result := Render(50, cfg)
		if len(result) == 0 {
			t.Errorf("Theme %q produced empty render", name)
		}
	}
}
