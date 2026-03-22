package progressbar

import (
	"strings"
	"testing"
)

func TestRenderZero(t *testing.T) {
	result := Render(0, 20)
	if !strings.Contains(result, strings.Repeat("░", 20)) {
		t.Error("expected all empty chars at 0%")
	}
}

func TestRenderFull(t *testing.T) {
	result := Render(1.0, 20)
	if !strings.Contains(result, strings.Repeat("█", 20)) {
		t.Error("expected all filled chars at 100%")
	}
}

func TestRenderHalf(t *testing.T) {
	result := Render(0.5, 20)
	filled := strings.Count(result, "█")
	empty := strings.Count(result, "░")
	if filled != 10 || empty != 10 {
		t.Errorf("expected 10 filled + 10 empty, got %d + %d", filled, empty)
	}
}

func TestRenderClampOver(t *testing.T) {
	result := Render(1.5, 10)
	if !strings.Contains(result, strings.Repeat("█", 10)) {
		t.Error("expected all filled when fraction > 1.0")
	}
}

func TestRenderClampNegative(t *testing.T) {
	result := Render(-0.5, 10)
	if !strings.Contains(result, strings.Repeat("░", 10)) {
		t.Error("expected all empty when fraction < 0")
	}
}

func TestRenderCharCount(t *testing.T) {
	result := Render(0.3, 15)
	filled := strings.Count(result, "█")
	empty := strings.Count(result, "░")
	if filled+empty != 15 {
		t.Errorf("expected 15 total chars, got %d", filled+empty)
	}
}
