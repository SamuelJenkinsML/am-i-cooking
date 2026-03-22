# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
make build        # Build binary → ./cook
make test         # Run all tests: go test ./...
make install      # Install to $GOPATH/bin
go test ./internal/calc/   # Run tests for a single package
```

Version info is injected via ldflags at build time (see `Makefile` and `internal/version/version.go`).

## Development Process

Use TDD for all changes. Write failing tests first, then implement the minimum code to make them pass, then refactor. Do not write implementation code without a corresponding test already in place.

## Architecture

This is a Bubble Tea TUI app that monitors Claude Code token usage in real time by tailing JSONL session logs from `~/.claude/projects/`.

**Data flow:** `watcher` → `parser` → `model` → `calc` → `gauge` → screen

- **`cmd/root.go`** — Cobra CLI entry point. Parses flags (`--project`, `--all`, `--window`), resolves the project directory, sets up the watcher, and starts the Bubble Tea program.
- **`internal/parser/`** — Discovers and reads JSONL files from Claude's project directory. Supports incremental parsing via file offsets so only new bytes are read on each change. Handles both main session files and subagent files.
- **`internal/watcher/`** — Uses fsnotify to watch for JSONL file changes. Sends `FileChangedMsg` to the Bubble Tea program on writes. Also runs a periodic 10-second rescan (`RescanTickMsg`) to catch new files.
- **`internal/calc/`** — Computes metrics from usage records: current/sustained/overall burn rates (weighted tokens/min), model breakdown percentages, cost estimates, and the verdict string. Uses a log-scale mapping for the gauge percentage.
- **`internal/gauge/`** — Renders a semicircular ASCII gauge using a character grid with lipgloss styling. The arc, tick marks, needle, and colors are all computed from the gauge percentage.
- **`internal/model/`** — The Bubble Tea model. Orchestrates the event loop: handles key presses, window resizes, parsed records, animation ticks (50ms ease-out for needle), and file change events. Composes the final view with lipgloss.

**Key design details:**
- Token weighting: output tokens 5x, cache creation 1.25x, cache read 0.1x, input 1x (reflects relative API pricing).
- Gauge scale is logarithmic: 1k tok/min → 25%, 10k → 50%, 100k → 75%, 1M → 100%.
- Needle animates with ease-out (15% of remaining distance per 50ms frame).
- Records are deduplicated by session ID + timestamp + model on each update.
