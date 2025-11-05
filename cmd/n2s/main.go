package main

import (
	"fmt"
	"os"

	"github.com/shubhamrasal/n2s/internal/app"
	"github.com/spf13/cobra"
)

var (
	// Version information (set by goreleaser)
	version = "dev"
	commit  = "none"
	date    = "unknown"

	natsURL    string
	configPath string
	readOnly   bool
)

var rootCmd = &cobra.Command{
	Use:   "n2s",
	Short: "Interactive NATS JetStream TUI",
	Long:  `A k9s-style terminal UI for managing NATS JetStream streams and consumers`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return app.Run(natsURL, configPath, readOnly)
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("n2s version %s (commit: %s, built: %s)\n", version, commit, date)
	},
}

func init() {
	rootCmd.Flags().StringVarP(&natsURL, "server", "s", "", "NATS server URL (overrides config file)")
	rootCmd.Flags().StringVarP(&configPath, "config", "c", "", "Config file path")
	rootCmd.Flags().BoolVarP(&readOnly, "read-only", "r", false, "Read-only mode (no deletions)")

	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
