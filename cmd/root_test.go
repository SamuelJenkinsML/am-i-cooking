package cmd

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/SamuelJenkinsML/am-i-cooking/internal/calc"
)

func TestMetricsToJSON_ZeroValue(t *testing.T) {
	m := calc.Metrics{}
	data := metricsToJSON(m)

	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify required fields exist
	requiredFields := []string{
		"tokens_used", "weighted_tokens", "burn_rate",
		"sustained_rate", "overall_rate", "estimated_cost",
		"window_elapsed_seconds", "window_size_seconds",
		"gauge_percent", "verdict", "models",
	}
	for _, field := range requiredFields {
		if _, ok := parsed[field]; !ok {
			t.Errorf("missing field %q in JSON output", field)
		}
	}

	// Zero metrics should produce "stone cold" or empty verdict
	if v, ok := parsed["burn_rate"].(float64); !ok || v != 0 {
		t.Errorf("expected burn_rate 0, got %v", parsed["burn_rate"])
	}
}

func TestMetricsToJSON_PopulatedMetrics(t *testing.T) {
	m := calc.Metrics{
		CurrentRate:         5432.1,
		SustainedRate:       3210.0,
		OverallRate:         2100.5,
		TotalWeightedTokens: 128750.5,
		TotalRawTokens:      42350,
		WindowElapsed:       2 * time.Hour,
		WindowSize:          5 * time.Hour,
		EstimatedCost:       0.42,
		GaugePercent:        43.2,
		Verdict:             "simmering nicely",
		OpusPercent:         60.0,
		SonnetPercent:       30.0,
		HaikuPercent:        10.0,
	}
	data := metricsToJSON(m)

	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if v := parsed["tokens_used"].(float64); int(v) != 42350 {
		t.Errorf("expected tokens_used 42350, got %v", v)
	}
	if v := parsed["burn_rate"].(float64); v != 5432.1 {
		t.Errorf("expected burn_rate 5432.1, got %v", v)
	}
	if v := parsed["verdict"].(string); v != "simmering nicely" {
		t.Errorf("expected verdict 'simmering nicely', got %q", v)
	}
	if v := parsed["window_elapsed_seconds"].(float64); v != 7200 {
		t.Errorf("expected window_elapsed_seconds 7200, got %v", v)
	}
	if v := parsed["estimated_cost"].(float64); v != 0.42 {
		t.Errorf("expected estimated_cost 0.42, got %v", v)
	}

	models := parsed["models"].(map[string]interface{})
	if v := models["opus_percent"].(float64); v != 60.0 {
		t.Errorf("expected opus_percent 60, got %v", v)
	}
}

func TestMetricsToJSON_Deterministic(t *testing.T) {
	m := calc.Metrics{
		CurrentRate: 1000,
		Verdict:     "test",
		WindowSize:  5 * time.Hour,
	}

	b1, _ := json.Marshal(metricsToJSON(m))
	b2, _ := json.Marshal(metricsToJSON(m))

	if string(b1) != string(b2) {
		t.Error("JSON output is not deterministic")
	}
}

func TestMutualExclusivity(t *testing.T) {
	// Both flags set should error
	if err := validateFlags(true, true); err == nil {
		t.Error("expected error when both --json and --once are set")
	}

	// Individual flags should not error
	if err := validateFlags(true, false); err != nil {
		t.Errorf("unexpected error for --json only: %v", err)
	}
	if err := validateFlags(false, true); err != nil {
		t.Errorf("unexpected error for --once only: %v", err)
	}
	if err := validateFlags(false, false); err != nil {
		t.Errorf("unexpected error for neither flag: %v", err)
	}
}
