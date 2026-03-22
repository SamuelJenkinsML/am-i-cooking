package gauge

import (
	"strings"
	"testing"
)

func TestRenderNotEmpty(t *testing.T) {
	result := Render(50)
	if len(result) == 0 {
		t.Error("Expected non-empty gauge render")
	}
}

func TestRenderContainsArcChars(t *testing.T) {
	result := Render(50)

	hasArc := strings.ContainsAny(result, "━┃╱╲")
	if !hasArc {
		t.Error("Expected arc characters in gauge render")
	}
}

func TestRenderContainsPivot(t *testing.T) {
	result := Render(50)
	if !strings.Contains(result, "●") {
		t.Error("Expected pivot character in gauge render")
	}
}

func TestRenderContainsNeedle(t *testing.T) {
	result := Render(75)
	hasNeedle := strings.ContainsAny(result, "─│╲╱")
	if !hasNeedle {
		t.Error("Expected needle characters in gauge render")
	}
}

func TestRenderZeroPercent(t *testing.T) {
	result := Render(0)
	if len(result) == 0 {
		t.Error("Expected non-empty gauge at 0%")
	}
}

func TestRenderHundredPercent(t *testing.T) {
	result := Render(100)
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
