package parser

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEncodePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/Users/foo/bar", "-Users-foo-bar"},
		{"/Users/samueljenkins/Projects/am-i-cooking", "-Users-samueljenkins-Projects-am-i-cooking"},
		{"/", "-"},
	}

	for _, tt := range tests {
		got := EncodePath(tt.input)
		if got != tt.expected {
			t.Errorf("EncodePath(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestParseFile(t *testing.T) {
	// Create a temporary JSONL file with test data
	dir := t.TempDir()
	path := filepath.Join(dir, "test-session.jsonl")

	now := time.Now().UTC()
	ts := now.Format(time.RFC3339Nano)
	oldTs := now.Add(-10 * time.Hour).Format(time.RFC3339Nano)

	content := `{"type":"file-history-snapshot","messageId":"abc"}
{"type":"user","message":{"role":"user","content":[{"type":"text","text":"hello"}]}}
{"type":"assistant","timestamp":"` + ts + `","sessionId":"sess1","message":{"model":"claude-opus-4-6","usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":200,"cache_read_input_tokens":300}}}
{"type":"assistant","timestamp":"` + oldTs + `","sessionId":"sess1","message":{"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":10,"output_tokens":5,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	records, offset, err := ParseFile(path, 0, 5*time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	if offset == 0 {
		t.Error("Expected non-zero offset after parsing")
	}

	// Should only get 1 record (the old one is outside 5h window)
	if len(records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records))
	}

	r := records[0]
	if r.Model != "claude-opus-4-6" {
		t.Errorf("Expected model claude-opus-4-6, got %s", r.Model)
	}
	if r.InputTokens != 100 {
		t.Errorf("Expected 100 input tokens, got %d", r.InputTokens)
	}
	if r.OutputTokens != 50 {
		t.Errorf("Expected 50 output tokens, got %d", r.OutputTokens)
	}
	if r.CacheCreationTokens != 200 {
		t.Errorf("Expected 200 cache creation tokens, got %d", r.CacheCreationTokens)
	}
	if r.CacheReadTokens != 300 {
		t.Errorf("Expected 300 cache read tokens, got %d", r.CacheReadTokens)
	}
	if r.SessionID != "sess1" {
		t.Errorf("Expected session ID sess1, got %s", r.SessionID)
	}
}

func TestParseFileIncremental(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-session.jsonl")

	now := time.Now().UTC()
	ts1 := now.Add(-1 * time.Minute).Format(time.RFC3339Nano)

	line1 := `{"type":"assistant","timestamp":"` + ts1 + `","sessionId":"s1","message":{"model":"claude-opus-4-6","usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}` + "\n"

	if err := os.WriteFile(path, []byte(line1), 0644); err != nil {
		t.Fatal(err)
	}

	// First parse
	records1, offset1, err := ParseFile(path, 0, 5*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if len(records1) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records1))
	}

	// Append a new line
	ts2 := now.Format(time.RFC3339Nano)
	line2 := `{"type":"assistant","timestamp":"` + ts2 + `","sessionId":"s1","message":{"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":200,"output_tokens":100,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}` + "\n"

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(line2)
	f.Close()

	// Incremental parse from offset
	records2, _, err := ParseFile(path, offset1, 5*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if len(records2) != 1 {
		t.Fatalf("Expected 1 new record, got %d", len(records2))
	}
	if records2[0].InputTokens != 200 {
		t.Errorf("Expected 200 input tokens, got %d", records2[0].InputTokens)
	}
}

func TestExtractSessionID(t *testing.T) {
	got := extractSessionID("/some/path/abc-123.jsonl")
	if got != "abc-123" {
		t.Errorf("Expected abc-123, got %s", got)
	}
}
