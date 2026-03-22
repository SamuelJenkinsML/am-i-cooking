package watcher

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewMultipleDirs(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	w, err := New([]string{dir1, dir2})
	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}
	defer w.Close()

	if len(w.projectDirs) != 2 {
		t.Errorf("expected 2 projectDirs, got %d", len(w.projectDirs))
	}
}

func TestNewSingleDir(t *testing.T) {
	dir := t.TempDir()

	w, err := New([]string{dir})
	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}
	defer w.Close()

	if len(w.projectDirs) != 1 {
		t.Errorf("expected 1 projectDir, got %d", len(w.projectDirs))
	}
}

func TestNewEmptySlice(t *testing.T) {
	_, err := New([]string{})
	if err == nil {
		t.Fatal("expected error for empty slice, got nil")
	}
}

func TestNewWithNonexistentDir(t *testing.T) {
	// Should not panic even if the directory doesn't exist.
	w, err := New([]string{"/tmp/nonexistent-watcher-test-dir-xyz"})
	// We don't assert on err — it may or may not error.
	// Just ensure no panic occurred.
	if err == nil && w != nil {
		w.Close()
	}
}

func TestNewWatchesSubagentDirs(t *testing.T) {
	dir := t.TempDir()

	// Create a session dir with a subagents subdirectory
	sessionDir := filepath.Join(dir, "session-abc")
	subagentDir := filepath.Join(sessionDir, "subagents")
	if err := os.MkdirAll(subagentDir, 0o755); err != nil {
		t.Fatal(err)
	}

	w, err := New([]string{dir})
	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}
	defer w.Close()

	// The watcher should have added the main dir, the session dir,
	// and the subagents dir. We can't easily inspect fsnotify's watched
	// list, but we verify no error was returned.
}
