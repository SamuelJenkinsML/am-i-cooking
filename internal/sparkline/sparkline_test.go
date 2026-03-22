package sparkline

import (
	"testing"
)

func TestBufferAdd(t *testing.T) {
	b := NewBuffer(10)
	b.Add(100)
	b.Add(200)
	samples := b.Samples()
	if len(samples) != 2 {
		t.Errorf("expected 2 samples, got %d", len(samples))
	}
	if samples[0] != 100 || samples[1] != 200 {
		t.Errorf("unexpected samples: %v", samples)
	}
}

func TestBufferRollover(t *testing.T) {
	b := NewBuffer(3)
	b.Add(1)
	b.Add(2)
	b.Add(3)
	b.Add(4)
	samples := b.Samples()
	if len(samples) != 3 {
		t.Errorf("expected 3 samples, got %d", len(samples))
	}
	if samples[0] != 2 || samples[1] != 3 || samples[2] != 4 {
		t.Errorf("expected [2 3 4], got %v", samples)
	}
}

func TestBufferEmpty(t *testing.T) {
	b := NewBuffer(10)
	samples := b.Samples()
	if len(samples) != 0 {
		t.Errorf("expected 0 samples, got %d", len(samples))
	}
}

func TestRenderEmpty(t *testing.T) {
	result := Render(nil, 10)
	if len(result) != 10 {
		t.Errorf("expected 10 chars, got %d", len([]rune(result)))
	}
}

func TestRenderAllZeros(t *testing.T) {
	samples := []float64{0, 0, 0, 0}
	result := Render(samples, 4)
	for _, r := range result {
		if r != ' ' {
			t.Errorf("expected spaces for all zeros, got %q", result)
			break
		}
	}
}

func TestRenderSinglePeak(t *testing.T) {
	samples := []float64{0, 0, 100, 0}
	result := Render(samples, 4)
	runes := []rune(result)
	if runes[2] != '█' {
		t.Errorf("expected peak at position 2, got %c", runes[2])
	}
}

func TestRenderScaling(t *testing.T) {
	samples := []float64{50, 100}
	result := Render(samples, 2)
	runes := []rune(result)
	// First should be roughly half, second should be max
	if runes[1] != '█' {
		t.Errorf("expected max block at position 1, got %c", runes[1])
	}
	// First should be less than max
	if runes[0] >= runes[1] {
		t.Errorf("expected first block to be less than second")
	}
}

func TestRenderWidthPadding(t *testing.T) {
	samples := []float64{100, 200}
	result := Render(samples, 5)
	runes := []rune(result)
	if len(runes) != 5 {
		t.Errorf("expected 5 runes, got %d", len(runes))
	}
	// First 3 should be padding (spaces)
	for i := 0; i < 3; i++ {
		if runes[i] != ' ' {
			t.Errorf("expected space at position %d, got %c", i, runes[i])
		}
	}
}

func TestRenderDownsample(t *testing.T) {
	// 6 samples into width 3 = average pairs
	samples := []float64{100, 200, 300, 400, 500, 600}
	result := Render(samples, 3)
	runes := []rune(result)
	if len(runes) != 3 {
		t.Errorf("expected 3 runes, got %d", len(runes))
	}
	// Last should be max (avg of 500+600 = 550, which is max)
	if runes[2] != '█' {
		t.Errorf("expected max block at last position, got %c", runes[2])
	}
}
