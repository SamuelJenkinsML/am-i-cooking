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

func TestDiscoverAllProjectDirs(t *testing.T) {
	baseDir := t.TempDir()

	// Create two subdirectories representing project dirs
	dir1 := filepath.Join(baseDir, "-Users-alice-projectA")
	dir2 := filepath.Join(baseDir, "-Users-bob-projectB")
	if err := os.MkdirAll(dir1, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir2, 0755); err != nil {
		t.Fatal(err)
	}

	// Also create a file (should be ignored)
	if err := os.WriteFile(filepath.Join(baseDir, "somefile.txt"), []byte("hi"), 0644); err != nil {
		t.Fatal(err)
	}

	dirs, err := DiscoverAllProjectDirs(baseDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(dirs) != 2 {
		t.Fatalf("Expected 2 dirs, got %d: %v", len(dirs), dirs)
	}

	// Check that both paths are present (order not guaranteed)
	found := map[string]bool{}
	for _, d := range dirs {
		found[d] = true
	}
	if !found[dir1] || !found[dir2] {
		t.Errorf("Expected dirs %s and %s, got %v", dir1, dir2, dirs)
	}
}

func TestDiscoverAllProjectDirs_Empty(t *testing.T) {
	baseDir := t.TempDir()

	dirs, err := DiscoverAllProjectDirs(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) != 0 {
		t.Fatalf("Expected 0 dirs, got %d", len(dirs))
	}
}

func TestDiscoverJSONLFilesInDir(t *testing.T) {
	projectDir := t.TempDir()

	// Create a main JSONL file
	mainFile := filepath.Join(projectDir, "session1.jsonl")
	if err := os.WriteFile(mainFile, []byte(`{}`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a subagent file
	subDir := filepath.Join(projectDir, "session1", "subagents")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	subFile := filepath.Join(subDir, "agent-1.jsonl")
	if err := os.WriteFile(subFile, []byte(`{}`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	files, err := DiscoverJSONLFilesInDir(projectDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 2 {
		t.Fatalf("Expected 2 files, got %d: %v", len(files), files)
	}

	found := map[string]bool{}
	for _, f := range files {
		found[f] = true
	}
	if !found[mainFile] || !found[subFile] {
		t.Errorf("Expected files %s and %s, got %v", mainFile, subFile, files)
	}
}

func TestParseAllProjects(t *testing.T) {
	baseDir := t.TempDir()

	now := time.Now().UTC()
	ts := now.Format(time.RFC3339Nano)

	// Project 1
	proj1 := filepath.Join(baseDir, "-Users-alice-projA")
	if err := os.MkdirAll(proj1, 0755); err != nil {
		t.Fatal(err)
	}
	file1 := filepath.Join(proj1, "sess1.jsonl")
	line1 := `{"type":"assistant","timestamp":"` + ts + `","sessionId":"sess1","message":{"model":"claude-sonnet-4-20250514","usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}` + "\n"
	if err := os.WriteFile(file1, []byte(line1), 0644); err != nil {
		t.Fatal(err)
	}

	// Project 2
	proj2 := filepath.Join(baseDir, "-Users-bob-projB")
	if err := os.MkdirAll(proj2, 0755); err != nil {
		t.Fatal(err)
	}
	file2 := filepath.Join(proj2, "sess2.jsonl")
	line2 := `{"type":"assistant","timestamp":"` + ts + `","sessionId":"sess2","message":{"model":"claude-sonnet-4-20250514","usage":{"input_tokens":200,"output_tokens":100,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}` + "\n"
	if err := os.WriteFile(file2, []byte(line2), 0644); err != nil {
		t.Fatal(err)
	}

	records, offsets, err := ParseAllProjects(baseDir, 5*time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	if len(records) != 2 {
		t.Fatalf("Expected 2 records, got %d", len(records))
	}

	if len(offsets) != 2 {
		t.Fatalf("Expected 2 offset entries, got %d", len(offsets))
	}

	if _, ok := offsets[file1]; !ok {
		t.Errorf("Expected offset entry for %s", file1)
	}
	if _, ok := offsets[file2]; !ok {
		t.Errorf("Expected offset entry for %s", file2)
	}

	// Verify we got records from both sessions
	sessions := map[string]bool{}
	for _, r := range records {
		sessions[r.SessionID] = true
	}
	if !sessions["sess1"] || !sessions["sess2"] {
		t.Errorf("Expected records from sess1 and sess2, got sessions: %v", sessions)
	}
}

func TestDiscoverAndTrackAll(t *testing.T) {
	baseDir := t.TempDir()

	// Create two project dirs with JSONL files
	proj1 := filepath.Join(baseDir, "-Users-alice-projA")
	if err := os.MkdirAll(proj1, 0755); err != nil {
		t.Fatal(err)
	}
	file1 := filepath.Join(proj1, "sess1.jsonl")
	if err := os.WriteFile(file1, []byte(`{}`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	proj2 := filepath.Join(baseDir, "-Users-bob-projB")
	if err := os.MkdirAll(proj2, 0755); err != nil {
		t.Fatal(err)
	}
	file2 := filepath.Join(proj2, "sess2.jsonl")
	if err := os.WriteFile(file2, []byte(`{}`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	offsets := make(map[string]int64)
	newFiles, err := DiscoverAndTrackAll(baseDir, offsets)
	if err != nil {
		t.Fatal(err)
	}

	if len(newFiles) != 2 {
		t.Fatalf("Expected 2 new files, got %d: %v", len(newFiles), newFiles)
	}

	if len(offsets) != 2 {
		t.Fatalf("Expected 2 offset entries, got %d", len(offsets))
	}

	// Call again — should find no new files
	newFiles2, err := DiscoverAndTrackAll(baseDir, offsets)
	if err != nil {
		t.Fatal(err)
	}
	if len(newFiles2) != 0 {
		t.Fatalf("Expected 0 new files on second call, got %d", len(newFiles2))
	}
}
