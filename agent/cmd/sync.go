package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ViP-GW2-Guilds/gorrik/agent/api"
	"github.com/ViP-GW2-Guilds/gorrik/agent/config"
	"github.com/ViP-GW2-Guilds/gorrik/agent/uploader"
	"github.com/ViP-GW2-Guilds/gorrik/agent/watcher"
	"github.com/spf13/cobra"
)

var (
	syncDryRun  bool
	syncSkipDps bool
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Catch-up: import new logs, then import and backfill dps.report URLs",
	Long: `Runs the full catch-up flow in order:

  1. import          — index any local logs not yet in the database
  2. import-dps-urls — copy dps.report URLs from the arcdps Log Manager cache
  3. backfill-dps    — upload anything still missing a dps.report URL

Steps 2 and 3 are skipped with --skip-dps. The log directory and Log Manager
cache path come from the config file, so no arguments are needed.`,
	Args: cobra.NoArgs,
	RunE: runSync,
}

func init() {
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "show what would happen without uploading or writing")
	syncCmd.Flags().BoolVar(&syncSkipDps, "skip-dps", false, "skip the dps.report import and backfill steps")
}

func runSync(cmd *cobra.Command, args []string) error {
	if err := config.Validate(cfg); err != nil && !syncDryRun {
		return err
	}

	dir, err := resolveLogDir(cfg)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		log.Println("sync interrupted")
		cancel()
	}()

	client := api.New(cfg, syncDryRun)

	// ── 1. import ─────────────────────────────────────────────────────────────
	log.Println("[1/3] indexing local logs...")
	up, err := uploader.New(cfg, syncDryRun)
	if err != nil {
		return fmt.Errorf("uploader: %w", err)
	}
	if err := watcher.BulkImport(ctx, dir, 4, cfg, up, client, syncDryRun); err != nil {
		return fmt.Errorf("import: %w", err)
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if syncSkipDps {
		log.Println("skipping dps.report steps (--skip-dps)")
		return nil
	}

	// ── 2. import-dps-urls ────────────────────────────────────────────────────
	if cache := resolveLogManagerCache(cfg); cache == "" {
		log.Println("[2/3] no arcdps Log Manager cache found — skipping dps.report URL import")
	} else {
		log.Printf("[2/3] importing dps.report URLs from %s", cache)
		if err := importDpsURLsFromCache(ctx, cache, client); err != nil {
			return fmt.Errorf("import-dps-urls: %w", err)
		}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// ── 3. backfill-dps ───────────────────────────────────────────────────────
	log.Println("[3/3] backfilling remaining dps.report uploads...")
	if err := backfillDpsReport(ctx, dir, 1.0, syncDryRun, client); err != nil {
		return fmt.Errorf("backfill-dps: %w", err)
	}
	return nil
}
