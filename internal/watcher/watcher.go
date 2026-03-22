package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
)

// FileChangedMsg is sent when a JSONL file is modified or created.
type FileChangedMsg struct {
	Path string
}

// RescanTickMsg triggers a periodic rescan for new files.
type RescanTickMsg struct{}

// Watcher monitors JSONL files for changes.
type Watcher struct {
	fsWatcher  *fsnotify.Watcher
	projectDirs []string
	program    *tea.Program
}

// New creates a new file watcher for the given project directories.
func New(projectDirs []string) (*Watcher, error) {
	if len(projectDirs) == 0 {
		return nil, fmt.Errorf("at least one project directory is required")
	}

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		fsWatcher:   fsw,
		projectDirs: projectDirs,
	}

	for _, projectDir := range projectDirs {
		// Watch the main project directory; skip if it fails (dir may not exist)
		if err := fsw.Add(projectDir); err != nil {
			continue
		}

		// Watch existing subagent directories
		entries, _ := os.ReadDir(projectDir)
		for _, e := range entries {
			if e.IsDir() {
				subDir := filepath.Join(projectDir, e.Name(), "subagents")
				if info, err := os.Stat(subDir); err == nil && info.IsDir() {
					fsw.Add(subDir)
				}
				// Also watch the session dir itself for new subagent dirs
				fsw.Add(filepath.Join(projectDir, e.Name()))
			}
		}
	}

	return w, nil
}

// SetProgram sets the Bubble Tea program for sending messages.
func (w *Watcher) SetProgram(p *tea.Program) {
	w.program = p
}

// Start begins listening for file events in a goroutine.
func (w *Watcher) Start() {
	go w.listen()
}

func (w *Watcher) listen() {
	for {
		select {
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}

			if event.Has(fsnotify.Write) && strings.HasSuffix(event.Name, ".jsonl") {
				if w.program != nil {
					w.program.Send(FileChangedMsg{Path: event.Name})
				}
			}

			if event.Has(fsnotify.Create) {
				// If a new directory is created, watch it for subagent files
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					w.fsWatcher.Add(event.Name)
					subDir := filepath.Join(event.Name, "subagents")
					if si, err := os.Stat(subDir); err == nil && si.IsDir() {
						w.fsWatcher.Add(subDir)
					}
				}

				// If a new JSONL file is created, notify
				if strings.HasSuffix(event.Name, ".jsonl") {
					if w.program != nil {
						w.program.Send(FileChangedMsg{Path: event.Name})
					}
				}
			}

		case _, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
		}
	}
}

// Close stops the watcher.
func (w *Watcher) Close() error {
	return w.fsWatcher.Close()
}

// RescanCmd returns a tea.Cmd that periodically triggers a rescan.
func RescanCmd() tea.Cmd {
	return tea.Tick(10*time.Second, func(t time.Time) tea.Msg {
		return RescanTickMsg{}
	})
}
