# am-i-cooking Roadmap

Post-v0.1.0 improvements organized into phases by theme. Tasks within each phase are independent and can be worked on in parallel unless noted.

---

## Phase 1: Visual Polish (complete)

- [x] **Responsive gauge** — Scale the arc/gauge to fit the current terminal width and height using `bubbletea`'s `WindowSizeMsg`. Redraw on resize events.
- [x] **Sparkline mini-chart** — Show burn rate over the last 30 minutes as a sparkline (▁▂▃▅▇) below the gauge. Rolling buffer of 60 samples at 30s intervals.
- [x] **Color theme system** — `--theme` flag with options: `default`, `minimal`, `neon`, `monochrome`. Each theme is a struct of color values applied globally via the model.
- [x] **Compact mode** — `--compact` flag for small terminals. Text-only 3-line layout. Auto-detects when terminal height < 20.
- [x] **Animated verdict transitions** — When the verdict changes, color blends smoothly over 500ms using Lab color space interpolation via `go-colorful`.
- [x] **Window elapsed progress bar** — Horizontal styled bar inline with the Window stats row showing progress through the `--window` duration.

## Phase 2: Missing Core Features (complete)

- [x] **`--all` flag** — Aggregate token usage across all projects in `~/.claude/projects/`. Walk the directory, sum JSONL entries, and display a combined burn rate. Ignore per-project filtering.
- [x] **Session breakdown view** — Toggle with `Tab` or `s` to switch between aggregate view and a per-session table. Each row shows session ID (truncated), token count, and rate. Uses bubbletea key handling.
- [x] **Keyboard shortcuts overlay** — Press `?` to show/hide a help overlay listing all keybindings. Render as a centered box on top of the main view.
- [x] **`--json` flag** — Single-shot JSON output to stdout, no TUI. Collect one snapshot of metrics and print `{"tokens_used": N, "burn_rate": N, "verdict": "..."}`. Exit immediately. Pipe-friendly.
- [x] **`--once` flag** — Render the TUI for a single frame, print it, and exit. Similar to `--json` but with the human-readable gauge output.

## Phase 3: Smarter Metrics

- **`--budget` flag** — Configurable monthly budget (default £90). Used by pacing and cost calculations. Store as a float in the model. Add to CLI via Cobra persistent flag.
- **Budget pacing** — Calculate expected spend at current burn rate vs days remaining in the billing cycle. Display "on track" / "over pace" / "under pace" with a percentage.
- **Peak burn rate tracking** — Track the highest instantaneous burn rate seen during the current session. Display as "Peak: X tok/min" alongside the current rate.
- **Activity heatmap** — Parse JSONL timestamps to build a 24-hour histogram of token usage. Render as a single row of block characters showing which hours are busiest.
- **Per-model cost breakdown** — Parse model IDs from JSONL entries. Apply per-model pricing (Opus, Sonnet, Haiku) to calculate actual £ cost, not just token count. Display as a small table.
- **Cache hit ratio display** — If JSONL entries contain cache read/write token fields, compute and display the cache hit ratio as a percentage.

## Phase 4: History & Persistence

- **SQLite local cache** — Store parsed JSONL data in a local SQLite database (`~/.am-i-cooking/history.db`). Schema: `sessions(id, project, start, end, tokens_in, tokens_out, cost)`. Populate on each run.
- **`history` subcommand** — `am-i-cooking history` shows a table of past sessions/days. Reads from SQLite. Support `--days N` to limit range.
- **Daily/weekly summary stats** — `am-i-cooking history --summary` shows aggregated stats: total tokens, total cost, average burn rate, most active day.
- **Export to CSV/JSON** — `am-i-cooking history --export csv` or `--export json` writes historical data to a file. Default filename includes date range.
- **Compare today vs yesterday** — `am-i-cooking history --compare` shows today's stats side-by-side with yesterday's. Highlight deltas with color (green = less spend, red = more).

## Phase 5: Notifications & Integrations

- **Terminal bell on threshold** — When burn rate exceeds a configurable threshold, trigger terminal bell (`\a`). Rate-limit to avoid spamming.
- **`--threshold` flag** — Set a burn rate threshold (tok/min). When exceeded, trigger the alert mechanism (bell, notification, or webhook). Works with all alert integrations.
- **Slack/Discord webhook** — `--webhook-url` flag. POST a JSON payload to the URL when threshold is exceeded. Include rate, project, and timestamp. Rate-limit to one alert per N minutes.
- **macOS menu bar mode** — `am-i-cooking --menubar` runs as a macOS menu bar app via a systray library. Shows current burn rate in the menu bar, click to expand stats.

## Phase 6: Distribution & Community

- **Homebrew formula** — Create a Homebrew tap (`homebrew-tap` repo) with a formula that installs the binary. Update goreleaser config to publish to the tap.
- **VHS demo tape** — Write a `.tape` file for [VHS](https://github.com/charmbracelet/vhs) that records an automated demo GIF. Add the GIF to README.
- **`--demo` flag** — Feed synthetic token data into the gauge for screenshots and demos. Generate realistic-looking burn rate patterns without needing real JSONL data.
- **Man page generation** — Use Cobra's `doc` package to generate a man page. Add a `make man` target and install instructions.
- **Shell completion docs** — Add a section to README with copy-paste instructions for bash, zsh, and fish completions using Cobra's built-in completion generation.
