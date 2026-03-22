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

func TestCalculateBySession_Empty(t *testing.T) {
	result := CalculateBySession(nil, 5*time.Hour)
	if result != nil {
		t.Errorf("Expected nil for empty records, got %v", result)
	}
}

func TestCalculateBySession_SingleSession(t *testing.T) {
	now := time.Now()
	records := []parser.UsageRecord{
		{
			Timestamp:    now.Add(-3 * time.Minute),
			Model:        "claude-opus-4-6",
			InputTokens:  1000,
			OutputTokens: 500,
			SessionID:    "sess1",
		},
		{
			Timestamp:    now.Add(-1 * time.Minute),
			Model:        "claude-opus-4-6",
			InputTokens:  800,
			OutputTokens: 400,
			SessionID:    "sess1",
		},
	}

	results := CalculateBySession(records, 5*time.Hour)
	if len(results) != 1 {
		t.Fatalf("Expected 1 session, got %d", len(results))
	}

	s := results[0]
	if s.SessionID != "sess1" {
		t.Errorf("Expected SessionID 'sess1', got %q", s.SessionID)
	}
	// Raw tokens: (1000+500) + (800+400) = 2700
	if s.TotalRawTokens != 2700 {
		t.Errorf("Expected TotalRawTokens 2700, got %d", s.TotalRawTokens)
	}
	if s.Rate <= 0 {
		t.Errorf("Expected Rate > 0, got %f", s.Rate)
	}
	if s.PrimaryModel != "claude-opus-4-6" {
		t.Errorf("Expected PrimaryModel 'claude-opus-4-6', got %q", s.PrimaryModel)
	}
}

func TestCalculateBySession_MultipleSessions(t *testing.T) {
	now := time.Now()
	records := []parser.UsageRecord{
		{
			Timestamp:    now.Add(-4 * time.Minute),
			Model:        "claude-opus-4-6",
			InputTokens:  1000,
			OutputTokens: 500,
			SessionID:    "sess1",
		},
		{
			Timestamp:    now.Add(-2 * time.Minute),
			Model:        "claude-opus-4-6",
			InputTokens:  800,
			OutputTokens: 400,
			SessionID:    "sess1",
		},
		{
			Timestamp:    now.Add(-1 * time.Minute),
			Model:        "claude-sonnet-4-20250514",
			InputTokens:  200,
			OutputTokens: 100,
			SessionID:    "sess2",
		},
	}

	results := CalculateBySession(records, 5*time.Hour)
	if len(results) != 2 {
		t.Fatalf("Expected 2 sessions, got %d", len(results))
	}

	// Check both sessions are present
	found := map[string]bool{}
	for _, s := range results {
		found[s.SessionID] = true
	}
	if !found["sess1"] || !found["sess2"] {
		t.Errorf("Expected both sess1 and sess2, got %v", found)
	}

	// Check per-session totals
	for _, s := range results {
		switch s.SessionID {
		case "sess1":
			if s.TotalRawTokens != 2700 {
				t.Errorf("sess1: Expected TotalRawTokens 2700, got %d", s.TotalRawTokens)
			}
		case "sess2":
			if s.TotalRawTokens != 300 {
				t.Errorf("sess2: Expected TotalRawTokens 300, got %d", s.TotalRawTokens)
			}
		}
	}
}

func TestCalculateBySession_SortedByRate(t *testing.T) {
	now := time.Now()
	records := []parser.UsageRecord{
		{
			Timestamp:    now.Add(-4 * time.Minute),
			Model:        "claude-opus-4-6",
			InputTokens:  100,
			OutputTokens: 10,
			SessionID:    "slow-sess",
		},
		{
			Timestamp:    now.Add(-2 * time.Minute),
			Model:        "claude-opus-4-6",
			InputTokens:  5000,
			OutputTokens: 10000,
			SessionID:    "fast-sess",
		},
	}

	results := CalculateBySession(records, 5*time.Hour)
	if len(results) != 2 {
		t.Fatalf("Expected 2 sessions, got %d", len(results))
	}

	if results[0].SessionID != "fast-sess" {
		t.Errorf("Expected first result to be 'fast-sess' (highest rate), got %q", results[0].SessionID)
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
