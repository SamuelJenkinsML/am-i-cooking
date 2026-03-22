package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/SamuelJenkinsML/am-i-cooking/internal/calc"
	"github.com/SamuelJenkinsML/am-i-cooking/internal/model"
	"github.com/SamuelJenkinsML/am-i-cooking/internal/parser"
	"github.com/SamuelJenkinsML/am-i-cooking/internal/theme"
	"github.com/SamuelJenkinsML/am-i-cooking/internal/version"
	"github.com/SamuelJenkinsML/am-i-cooking/internal/watcher"
)

var (
	projectPath string
	allProjects bool
	windowStr   string
	themeName   string
	compact     bool
	jsonFlag    bool
	onceFlag    bool
)

var rootCmd = &cobra.Command{
	Use:   "am-i-cooking",
	Short: "Real-time terminal gauge for your Claude Code token burn rate",
	Long: `am-i-cooking monitors your Claude Code token usage in real time,
displaying a beautiful animated gauge that shows how hard you're cooking.

It reads JSONL session logs from ~/.claude/projects/ and calculates
burn rates, cost estimates, and model breakdowns over a rolling window.`,
	Version: version.Version,
	RunE:    run,
}

func init() {
	rootCmd.Flags().StringVarP(&projectPath, "project", "p", "", "project path to monitor (default: current directory)")
	rootCmd.Flags().BoolVarP(&allProjects, "all", "a", false, "monitor all projects")
	rootCmd.Flags().StringVarP(&windowStr, "window", "w", "5h", "rolling window duration (e.g. 5h, 2h30m)")
	rootCmd.Flags().StringVarP(&themeName, "theme", "t", "default", "color theme (default, minimal, neon, monochrome)")
	rootCmd.Flags().BoolVar(&compact, "compact", false, "compact text-only mode (auto-detects small terminals)")
	rootCmd.Flags().BoolVar(&jsonFlag, "json", false, "output metrics as JSON and exit")
	rootCmd.Flags().BoolVar(&onceFlag, "once", false, "render a single frame and exit")

	rootCmd.SetVersionTemplate(fmt.Sprintf("am-i-cooking %s (commit: %s, built: %s)\n",
		version.Version, version.Commit, version.Date))
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

type jsonOutput struct {
	TokensUsed         int        `json:"tokens_used"`
	WeightedTokens     float64    `json:"weighted_tokens"`
	BurnRate           float64    `json:"burn_rate"`
	SustainedRate      float64    `json:"sustained_rate"`
	OverallRate        float64    `json:"overall_rate"`
	EstimatedCost      float64    `json:"estimated_cost"`
	WindowElapsedSecs  float64    `json:"window_elapsed_seconds"`
	WindowSizeSecs     float64    `json:"window_size_seconds"`
	GaugePercent       float64    `json:"gauge_percent"`
	Verdict            string     `json:"verdict"`
	Models             jsonModels `json:"models"`
}

type jsonModels struct {
	OpusPercent   float64 `json:"opus_percent"`
	SonnetPercent float64 `json:"sonnet_percent"`
	HaikuPercent  float64 `json:"haiku_percent"`
}

func metricsToJSON(m calc.Metrics) jsonOutput {
	return jsonOutput{
		TokensUsed:        m.TotalRawTokens,
		WeightedTokens:    m.TotalWeightedTokens,
		BurnRate:          m.CurrentRate,
		SustainedRate:     m.SustainedRate,
		OverallRate:       m.OverallRate,
		EstimatedCost:     m.EstimatedCost,
		WindowElapsedSecs: m.WindowElapsed.Seconds(),
		WindowSizeSecs:    m.WindowSize.Seconds(),
		GaugePercent:      m.GaugePercent,
		Verdict:           m.Verdict,
		Models: jsonModels{
			OpusPercent:   m.OpusPercent,
			SonnetPercent: m.SonnetPercent,
			HaikuPercent:  m.HaikuPercent,
		},
	}
}

func validateFlags(jsonF, onceF bool) error {
	if jsonF && onceF {
		return fmt.Errorf("--json and --once are mutually exclusive")
	}
	return nil
}

func printJSON(m calc.Metrics) error {
	data := metricsToJSON(m)
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling JSON: %w", err)
	}
	fmt.Println(string(b))
	return nil
}

func run(cmd *cobra.Command, args []string) error {
	if err := validateFlags(jsonFlag, onceFlag); err != nil {
		return err
	}

	window, err := time.ParseDuration(windowStr)
	if err != nil {
		return fmt.Errorf("invalid window duration: %w", err)
	}

	t, err := theme.ByName(themeName)
	if err != nil {
		return err
	}

	var watchDirs []string
	pp := projectPath

	if allProjects {
		dirs, err := parser.DiscoverAllProjectDirs("")
		if err != nil {
			return fmt.Errorf("discovering projects: %w", err)
		}
		if len(dirs) == 0 {
			return fmt.Errorf("no Claude Code project data found in ~/.claude/projects/")
		}
		watchDirs = dirs

		if jsonFlag {
			records, _, err := parser.ParseAllProjects("", window)
			if err != nil {
				return fmt.Errorf("parsing records: %w", err)
			}
			metrics := calc.Calculate(records, window)
			return printJSON(metrics)
		}

		if onceFlag {
			records, _, err := parser.ParseAllProjects("", window)
			if err != nil {
				return fmt.Errorf("parsing records: %w", err)
			}
			metrics := calc.Calculate(records, window)
			m := model.NewSnapshot(metrics, window, t, compact)
			fmt.Print(m.View())
			return nil
		}
	} else {
		if pp == "" {
			pp, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("getting working directory: %w", err)
			}
		}

		projectDir, err := parser.ProjectDir(pp)
		if err != nil {
			return fmt.Errorf("resolving project dir: %w", err)
		}

		if _, err := os.Stat(projectDir); os.IsNotExist(err) {
			return fmt.Errorf("no Claude Code data found for this project.\nExpected: %s\nRun some Claude Code sessions first", projectDir)
		}

		if jsonFlag {
			records, _, err := parser.ParseAll(pp, window)
			if err != nil {
				return fmt.Errorf("parsing records: %w", err)
			}
			metrics := calc.Calculate(records, window)
			return printJSON(metrics)
		}

		if onceFlag {
			records, _, err := parser.ParseAll(pp, window)
			if err != nil {
				return fmt.Errorf("parsing records: %w", err)
			}
			metrics := calc.Calculate(records, window)
			m := model.NewSnapshot(metrics, window, t, compact)
			fmt.Print(m.View())
			return nil
		}

		watchDirs = []string{projectDir}
	}

	w, err := watcher.New(watchDirs)
	if err != nil {
		return fmt.Errorf("setting up file watcher: %w", err)
	}
	defer w.Close()

	m := model.New(pp, window, allProjects, t, compact)
	p := tea.NewProgram(m, tea.WithAltScreen())

	w.SetProgram(p)
	w.Start()

	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
