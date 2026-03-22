# am-i-cooking

Real-time terminal gauge for your Claude Code token burn rate.

**am-i-cooking** monitors your Claude Code session logs and displays a live animated gauge showing how fast you're burning through tokens — with burn rates, cost estimates, and model breakdowns.

<!-- TODO: Add screenshot/demo GIF -->

## Installation

### Go install

```bash
go install github.com/SamuelJenkinsML/am-i-cooking@latest
```

### Download binary

Grab the latest release from the [Releases page](https://github.com/SamuelJenkinsML/am-i-cooking/releases).

## Usage

```bash
# Monitor current project directory
am-i-cooking

# Monitor a specific project
am-i-cooking --project /path/to/project

# Use a different rolling window
am-i-cooking --window 2h30m

# Monitor all projects
am-i-cooking --all

# Use a different color theme
am-i-cooking --theme neon

# Compact text-only mode (also auto-detects small terminals)
am-i-cooking --compact

# Single-shot JSON output (pipe-friendly)
am-i-cooking --json

# Render a single frame and exit
am-i-cooking --once
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--project` | `-p` | current dir | Project path to monitor |
| `--all` | `-a` | `false` | Monitor all projects |
| `--window` | `-w` | `5h` | Rolling window duration |
| `--theme` | `-t` | `default` | Color theme (`default`, `minimal`, `neon`, `monochrome`) |
| `--compact` | | `false` | Compact text-only mode (auto-detects small terminals) |
| `--json` | | `false` | Output metrics as JSON and exit |
| `--once` | | `false` | Render a single frame and exit |
| `--version` | `-v` | | Print version info |
| `--help` | `-h` | | Print help |

## How it works

Claude Code writes JSONL session logs to `~/.claude/projects/`. **am-i-cooking** reads these logs, parses token usage from each assistant response, and calculates:

- **Burn rate** — weighted tokens/min over the last 5 minutes
- **Sustained rate** — weighted tokens/min over the last 30 minutes
- **Cost estimate** — approximate spend for the current window
- **Model breakdown** — percentage split across Opus, Sonnet, and Haiku

Token weights reflect relative API pricing: output tokens count 5x, cache creation 1.25x, cache reads 0.1x, and input tokens 1x.

The gauge updates live via filesystem watching — no polling required.

### Display features

- **Responsive gauge** — the semicircular arc scales to fit your terminal and redraws on resize
- **Sparkline** — a mini burn-rate chart showing the last 30 minutes of activity
- **Color themes** — four built-in themes: `default`, `minimal`, `neon`, `monochrome`
- **Compact mode** — a 3-line text-only layout for small terminals or tmux panes
- **Animated transitions** — the needle eases smoothly and verdict colors blend on change
- **Progress bar** — shows how far through the rolling window you are
- **Help overlay** — press `?` to show/hide keyboard shortcuts

## Building from source

```bash
git clone https://github.com/SamuelJenkinsML/am-i-cooking.git
cd am-i-cooking
make build
```

## Contributing

1. Fork the repo
2. Create a feature branch (`git checkout -b my-feature`)
3. Commit your changes
4. Push to your branch and open a PR

## License

[MIT](LICENSE)
