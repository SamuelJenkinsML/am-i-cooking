package calc

import (
	"testing"
	"time"

	"github.com/SamuelJenkinsML/am-i-cooking/internal/parser"
)

func TestWeightedTokens(t *testing.T) {
	r := parser.UsageRecord{
		InputTokens:         100,
		OutputTokens:        50,
		CacheCreationTokens: 200,
		CacheReadTokens:     300,
	}

	got := WeightedTokens(r)
	// 100*1 + 50*5 + 200*1.25 + 300*0.1 = 100 + 250 + 250 + 30 = 630
	expected := 630.0
	if got != expected {
		t.Errorf("WeightedTokens = %f, want %f", got, expected)
	}
}

func TestRawTokens(t *testing.T) {
	r := parser.UsageRecord{
		InputTokens:         100,
		OutputTokens:        50,
		CacheCreationTokens: 200,
		CacheReadTokens:     300,
	}

	got := RawTokens(r)
	if got != 650 {
		t.Errorf("RawTokens = %d, want 650", got)
	}
}

func TestCalculateEmpty(t *testing.T) {
	m := Calculate(nil, 5*time.Hour)

	if m.Verdict != "stone cold... get cooking!" {
		t.Errorf("Expected stone cold verdict, got %q", m.Verdict)
	}
	if m.GaugePercent != 0 {
		t.Errorf("Expected 0%% gauge, got %f", m.GaugePercent)
	}
	if m.WindowRemaining != 5*time.Hour {
		t.Errorf("Expected 5h remaining, got %v", m.WindowRemaining)
	}
}

func TestCalculateWithRecords(t *testing.T) {
	now := time.Now()

	records := []parser.UsageRecord{
		{
			Timestamp:   now.Add(-2 * time.Minute),
			Model:       "claude-opus-4-6",
			InputTokens: 1000,
			OutputTokens: 500,
			CacheCreationTokens: 200,
			CacheReadTokens: 100,
		},
		{
			Timestamp:   now.Add(-1 * time.Minute),
			Model:       "claude-haiku-4-5-20251001",
			InputTokens: 500,
			OutputTokens: 200,
			CacheCreationTokens: 0,
			CacheReadTokens: 50,
		},
	}

	m := Calculate(records, 5*time.Hour)

	if m.TotalRawTokens == 0 {
		t.Error("Expected non-zero total raw tokens")
	}
	if m.CurrentRate == 0 {
		t.Error("Expected non-zero current rate")
	}
	if m.OpusPercent == 0 {
		t.Error("Expected non-zero Opus percentage")
	}
	if m.HaikuPercent == 0 {
		t.Error("Expected non-zero Haiku percentage")
	}
	if m.Verdict == "" {
		t.Error("Expected non-empty verdict")
	}
}

func TestRateToPercent(t *testing.T) {
	tests := []struct {
		rate     float64
		minPct   float64
		maxPct   float64
	}{
		{0, 0, 0},
		{1000, 24, 26},       // ~25%
		{10000, 49, 51},      // ~50%
		{100000, 74, 76},     // ~75%
		{1000000, 99, 100},   // ~100%
	}

	for _, tt := range tests {
		got := rateToPercent(tt.rate)
		if got < tt.minPct || got > tt.maxPct {
			t.Errorf("rateToPercent(%f) = %f, want between %f and %f", tt.rate, got, tt.minPct, tt.maxPct)
		}
	}
}
