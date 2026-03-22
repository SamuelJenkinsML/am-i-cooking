package parser

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// UsageRecord holds parsed token usage from a single assistant response.
type UsageRecord struct {
	Timestamp           time.Time
	Model               string
	InputTokens         int
	OutputTokens        int
	CacheCreationTokens int
	CacheReadTokens     int
	SessionID           string
	IsSubagent          bool
}

// jsonLine is the minimal structure we need from each JSONL line.
type jsonLine struct {
	Type      string    `json:"type"`
	Timestamp string    `json:"timestamp"`
	SessionID string    `json:"sessionId"`
	Message   *jMessage `json:"message"`
}

type jMessage struct {
	Model string `json:"model"`
	Usage jUsage `json:"usage"`
}

type jUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

// FileOffset tracks how far we've read into each file for incremental parsing.
type FileOffset struct {
	Path   string
	Offset int64
}

// EncodePath converts a filesystem path to Claude's project dir encoding.
// e.g. /Users/foo/bar → -Users-foo-bar
func EncodePath(p string) string {
	return strings.ReplaceAll(p, "/", "-")
}

// DiscoverJSONLFiles finds all JSONL session files for a given project path.
func DiscoverJSONLFiles(projectPath string) ([]string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	encoded := EncodePath(projectPath)
	projectDir := filepath.Join(homeDir, ".claude", "projects", encoded)

	var files []string

	// Main session files
	mainGlob, err := filepath.Glob(filepath.Join(projectDir, "*.jsonl"))
	if err != nil {
		return nil, err
	}
	files = append(files, mainGlob...)

	// Subagent files
	subGlob, err := filepath.Glob(filepath.Join(projectDir, "*/subagents/agent-*.jsonl"))
	if err != nil {
		return nil, err
	}
	files = append(files, subGlob...)

	return files, nil
}

// ProjectDir returns the Claude project directory for a given project path.
func ProjectDir(projectPath string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	encoded := EncodePath(projectPath)
	return filepath.Join(homeDir, ".claude", "projects", encoded), nil
}

// ParseFile reads a JSONL file from the given offset and returns new UsageRecords.
// It returns the updated offset for subsequent incremental reads.
func ParseFile(path string, offset int64, window time.Duration) ([]UsageRecord, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, offset, err
	}
	defer f.Close()

	if offset > 0 {
		if _, err := f.Seek(offset, 0); err != nil {
			return nil, offset, err
		}
	}

	isSubagent := strings.Contains(path, "/subagents/")
	sessionID := extractSessionID(path)
	cutoff := time.Now().Add(-window)

	var records []UsageRecord
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024) // 10MB max line

	for scanner.Scan() {
		line := scanner.Bytes()
		offset += int64(len(line)) + 1 // +1 for newline

		var jl jsonLine
		if err := json.Unmarshal(line, &jl); err != nil {
			continue
		}

		if jl.Type != "assistant" || jl.Message == nil {
			continue
		}

		ts, err := time.Parse(time.RFC3339Nano, jl.Timestamp)
		if err != nil {
			ts, err = time.Parse("2006-01-02T15:04:05.000Z", jl.Timestamp)
			if err != nil {
				continue
			}
		}

		if ts.Before(cutoff) {
			continue
		}

		sid := jl.SessionID
		if sid == "" {
			sid = sessionID
		}

		records = append(records, UsageRecord{
			Timestamp:           ts,
			Model:               jl.Message.Model,
			InputTokens:         jl.Message.Usage.InputTokens,
			OutputTokens:        jl.Message.Usage.OutputTokens,
			CacheCreationTokens: jl.Message.Usage.CacheCreationInputTokens,
			CacheReadTokens:     jl.Message.Usage.CacheReadInputTokens,
			SessionID:           sid,
			IsSubagent:          isSubagent,
		})
	}

	return records, offset, scanner.Err()
}

// ParseAll does a full parse of all discovered JSONL files for a project.
func ParseAll(projectPath string, window time.Duration) ([]UsageRecord, map[string]int64, error) {
	files, err := DiscoverJSONLFiles(projectPath)
	if err != nil {
		return nil, nil, err
	}

	offsets := make(map[string]int64)
	var allRecords []UsageRecord

	for _, f := range files {
		records, newOffset, err := ParseFile(f, 0, window)
		if err != nil {
			continue // skip files we can't read
		}
		offsets[f] = newOffset
		allRecords = append(allRecords, records...)
	}

	return allRecords, offsets, nil
}

// ParseIncremental re-reads only the new bytes from each tracked file.
func ParseIncremental(offsets map[string]int64, window time.Duration) ([]UsageRecord, error) {
	var allRecords []UsageRecord

	for path, off := range offsets {
		records, newOffset, err := ParseFile(path, off, window)
		if err != nil {
			continue
		}
		offsets[path] = newOffset
		allRecords = append(allRecords, records...)
	}

	return allRecords, nil
}

// DiscoverAndTrack finds any new JSONL files not already in offsets and adds them.
// Returns the list of newly discovered file paths.
func DiscoverAndTrack(projectPath string, offsets map[string]int64) ([]string, error) {
	files, err := DiscoverJSONLFiles(projectPath)
	if err != nil {
		return nil, err
	}

	var newFiles []string
	for _, f := range files {
		if _, exists := offsets[f]; !exists {
			offsets[f] = 0
			newFiles = append(newFiles, f)
		}
	}
	return newFiles, nil
}

func extractSessionID(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, ".jsonl")
}
