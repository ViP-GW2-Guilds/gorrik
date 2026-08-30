package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/ViP-GW2-Guilds/gorrik/agent/api"
	"github.com/ViP-GW2-Guilds/gorrik/agent/config"
	"github.com/ViP-GW2-Guilds/gorrik/agent/dpsreport"
	"github.com/spf13/cobra"
)

var (
	backfillDir    string
	backfillDryRun bool
	backfillRate   float64
)

var backfillDpsCmd = &cobra.Command{
	Use:   "backfill-dps",
	Short: "Upload logs missing a dps.report URL",
	Long: `Walks the local log directory and uploads to dps.report any log that
is in the database but has no dps_report_url set. Safe to interrupt and re-run.

HTTP 429 responses are handled automatically: the command waits for the
duration specified in the Retry-After header before retrying (up to 3 times).

Run 'gorrik import-dps-urls' first to populate URLs from the arcdps
Log Manager cache without re-uploading files.`,
	Args: cobra.NoArgs,
	RunE: runBackfillDps,
}

func init() {
	backfillDpsCmd.Flags().StringVar(&backfillDir, "dir", "", "log directory (defaults to arcdps log directory from config)")
	backfillDpsCmd.Flags().BoolVar(&backfillDryRun, "dry-run", false, "show what would be uploaded without actually uploading")
	backfillDpsCmd.Flags().Float64Var(&backfillRate, "rate", 1.0, "max uploads per second (e.g. 0.5 = one every two seconds)")
}

func runBackfillDps(cmd *cobra.Command, args []string) error {
	if backfillRate <= 0 {
		return fmt.Errorf("--rate must be greater than zero")
	}

	dir := backfillDir
	if dir == "" {
		if cfg.Arcdps.LogDir != "" {
			dir = cfg.Arcdps.LogDir
		} else if cfg.Arcdps.AutoDetect {
			dir = config.DetectArcdpsLogDir()
		}
	}
	if dir == "" {
		return cmd.Help()
	}
	if _, err := os.Stat(dir); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		log.Println("backfill interrupted")
		cancel()
	}()

	return backfillDpsReport(ctx, dir, backfillRate, backfillDryRun, api.New(cfg, backfillDryRun))
}

// backfillDpsReport uploads to dps.report every log under dir that the database
// still has no dps.report URL for, then patches the URL back.
func backfillDpsReport(ctx context.Context, dir string, rate float64, dryRun bool, client *api.Client) error {
	if rate <= 0 {
		return fmt.Errorf("rate must be greater than zero")
	}
	sleepBetween := time.Duration(float64(time.Second) / rate)

	log.Println("fetching logs without dps.report URL...")
	entries, err := client.FetchLogsWithoutDpsURL(ctx)
	if err != nil {
		return err
	}
	log.Printf("%d logs need a dps.report URL", len(entries))
	if len(entries) == 0 {
		return nil
	}

	// Build a filename → log ID map.
	need := make(map[string]string, len(entries))
	for _, e := range entries {
		need[e.Filename] = e.ID
	}

	// Walk local dir to find matching files.
	var toUpload []string
	err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if _, ok := need[filepath.Base(path)]; ok && isLogFile(path) {
			toUpload = append(toUpload, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	log.Printf("%d local files matched; uploading at %g/s (%s between uploads)",
		len(toUpload), rate, sleepBetween)

	dpsClient := dpsreport.New(cfg.Behaviour.DpsReportUserToken)

	uploaded, skipped, failed := 0, 0, 0
	for _, path := range toUpload {
		if ctx.Err() != nil {
			break
		}
		base := filepath.Base(path)
		logID := need[base]

		if dryRun {
			log.Printf("[dry-run] would upload %s → dps.report (log %s)", base, logID)
			skipped++
			continue
		}

		permalink, err := dpsClient.Upload(ctx, path)
		if err != nil {
			log.Printf("upload failed for %s: %v", base, err)
			failed++
			time.Sleep(sleepBetween)
			continue
		}

		if err := client.PatchLogDpsURL(ctx, logID, permalink); err != nil {
			log.Printf("PATCH failed for %s (%s): %v", base, logID, err)
			failed++
		} else {
			log.Printf("ok: %s → %s", base, permalink)
			uploaded++
		}

		time.Sleep(sleepBetween)
	}

	log.Printf("done: %d uploaded, %d skipped, %d failed", uploaded, skipped, failed)
	return nil
}

// isLogFile is duplicated from watcher to avoid coupling cmd → watcher internals.
func isLogFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".evtc") ||
		strings.HasSuffix(lower, ".evtc.zip") ||
		strings.HasSuffix(lower, ".zevtc")
}
