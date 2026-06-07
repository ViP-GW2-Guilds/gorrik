package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/ViP-GW2-Guilds/gorrik/agent/api"
	"github.com/ViP-GW2-Guilds/gorrik/agent/config"
	"github.com/ViP-GW2-Guilds/gorrik/agent/uploader"
	"github.com/ViP-GW2-Guilds/gorrik/agent/watcher"
)

var (
	importDryRun bool
	importDir    string
	importWorkers int
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Bulk-import all existing logs from a directory",
	Long: `Walks the given directory (or the configured log directory) and processes
every .evtc / .zevtc file found. Uploads are parallelised across --workers
goroutines; parsing is fast, so workers are sized for upload throughput.

Existing logs that have already been indexed are not re-uploaded — the API
rejects duplicates based on filename.`,
	RunE: runImport,
}

func init() {
	importCmd.Flags().BoolVar(&importDryRun, "dry-run", false, "parse and log what would be uploaded without actually uploading")
	importCmd.Flags().StringVar(&importDir, "dir", "", "directory to import from (defaults to arcdps log directory from config)")
	importCmd.Flags().IntVar(&importWorkers, "workers", 4, "number of parallel upload workers")
}

func runImport(cmd *cobra.Command, args []string) error {
	if err := config.Validate(cfg); err != nil && !importDryRun {
		return err
	}

	dir := importDir
	if dir == "" {
		if cfg.Arcdps.LogDir != "" {
			dir = cfg.Arcdps.LogDir
		} else if cfg.Arcdps.AutoDetect {
			dir = config.DetectArcdpsLogDir()
		}
	}
	if dir == "" {
		return fmt.Errorf("no directory specified — use --dir or set arcdps.log_dir in config")
	}

	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("directory %s: %w", dir, err)
	}

	up, err := uploader.New(cfg, importDryRun)
	if err != nil {
		return fmt.Errorf("uploader: %w", err)
	}

	apiClient := api.New(cfg, importDryRun)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		log.Println("import interrupted, finishing in-flight uploads...")
		cancel()
	}()

	return watcher.BulkImport(ctx, dir, importWorkers, cfg, up, apiClient, importDryRun)
}
