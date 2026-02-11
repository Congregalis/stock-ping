package cmd

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/congregalis/stock-ping/config"
	"github.com/congregalis/stock-ping/notify"
	"github.com/congregalis/stock-ping/stock"
	"github.com/congregalis/stock-ping/tui"
	"github.com/congregalis/stock-ping/watcher"
)

// RunDashboard executes the dashboard subcommand with TUI
func RunDashboard(args []string) {
	fs := flag.NewFlagSet("dashboard", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: stock-ping dashboard\n\n")
		fmt.Fprintf(os.Stderr, "Launch interactive TUI dashboard for stock monitoring.\n")
		fmt.Fprintf(os.Stderr, "Supports hot-reload: edit ~/.stock-ping.yaml to update rules.\n\n")
		fmt.Fprintf(os.Stderr, "Keybindings:\n")
		fmt.Fprintf(os.Stderr, "  r       Refresh all stocks\n")
		fmt.Fprintf(os.Stderr, "  q       Quit\n")
		fmt.Fprintf(os.Stderr, "  ?       Help\n")
	}

	fs.Parse(args)

	// Load config
	configPath := config.DefaultConfigPath()
	cfg, err := config.LoadFrom(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if cfg.Finnhub.APIKey == "" {
		fmt.Fprintf(os.Stderr, "Error: Finnhub API key not configured.\n")
		fmt.Fprintf(os.Stderr, "Please add your API key to ~/.stock-ping.yaml\n")
		os.Exit(1)
	}

	if len(cfg.Rules) == 0 {
		fmt.Fprintf(os.Stderr, "No monitoring rules configured.\n")
		fmt.Fprintf(os.Stderr, "Add rules with: stock-ping config add --symbol AAPL --price-above 200\n")
		os.Exit(1)
	}

	// Create clients
	stockClient := stock.NewClient(cfg.Finnhub.APIKey)
	notifier := notify.NewNotifier(cfg.Bark.ServerURL, cfg.Bark.Key)

	// Create TUI model
	model := tui.NewModel(cfg, stockClient, notifier, configPath)

	// Create program
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Setup hot-reload watcher
	configWatcher, err := watcher.NewConfigWatcher(configPath, func(newCfg *config.Config) {
		p.Send(tui.ReloadConfig(newCfg)())
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to setup config watcher: %v\n", err)
	} else {
		configWatcher.Start()
		defer configWatcher.Stop()
	}

	// Run TUI
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
