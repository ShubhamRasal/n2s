package app

import (
	"fmt"

	"github.com/rivo/tview"
	"github.com/shubhamrasal/n2s/internal/config"
	"github.com/shubhamrasal/n2s/internal/nats"
	"github.com/shubhamrasal/n2s/internal/plugins"
	"github.com/shubhamrasal/n2s/internal/ui"
)

// Run starts the N9S application
func Run(serverURL, configPath string, readOnly bool) error {
	// Load configuration
	cfg, err := config.Load(configPath, serverURL)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize NATS client
	nc, err := nats.NewClient(cfg.CurrentContext())
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}
	defer nc.Close()

	// Initialize plugin manager
	pluginMgr := plugins.NewManager()
	if err := pluginMgr.LoadPlugins(); err != nil {
		// Log error but continue - plugins are optional
		fmt.Printf("Warning: Failed to load plugins: %v\n", err)
	}

	// Create tview application
	app := tview.NewApplication()

	// Initialize UI manager
	uiManager := ui.NewUIManager(app, nc, cfg, pluginMgr, readOnly)

	// Start the UI
	if err := uiManager.Start(); err != nil {
		return fmt.Errorf("failed to start UI: %w", err)
	}

	return nil
}

