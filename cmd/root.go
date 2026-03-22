package cmd

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

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

	rootCmd.SetVersionTemplate(fmt.Sprintf("am-i-cooking %s (commit: %s, built: %s)\n",
		version.Version, version.Commit, version.Date))
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	pp := projectPath
	if pp == "" {
		var err error
		pp, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
	}

	window, err := time.ParseDuration(windowStr)
	if err != nil {
		return fmt.Errorf("invalid window duration: %w", err)
	}

	t, err := theme.ByName(themeName)
	if err != nil {
		return err
	}

	projectDir, err := parser.ProjectDir(pp)
	if err != nil {
		return fmt.Errorf("resolving project dir: %w", err)
	}

	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return fmt.Errorf("no Claude Code data found for this project.\nExpected: %s\nRun some Claude Code sessions first", projectDir)
	}

	w, err := watcher.New(projectDir)
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
