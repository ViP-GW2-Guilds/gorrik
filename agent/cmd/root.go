package cmd

import (
	"fmt"
	"os"

	"github.com/ViP-GW2-Guilds/gorrik/agent/config"
	"github.com/spf13/cobra"
)

var (
	cfgPath string
	cfg     *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "gorrik",
	Short: "Gorrik — GW2 combat log archiver",
	Long: `Gorrik watches your arcdps log directory, parses new logs for metadata,
uploads the raw files to Cloudflare R2, and indexes them in your web app.

Run 'gorrik setup' on first use to create your configuration file.
Run 'gorrik' with no arguments to see current status.`,
	Args: cobra.NoArgs,
	RunE: runStatus,
}

// Execute is the entry point called from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", config.DefaultPath(), "path to gorrik.toml")
	cobra.OnInitialize(loadConfig)

	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(watchCmd)
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(reparseCmd)
	rootCmd.AddCommand(serviceCmd)
	rootCmd.AddCommand(backfillDpsCmd)
	rootCmd.AddCommand(importDpsUrlsCmd)
}

func loadConfig() {
	var err error
	cfg, err = config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}
}
