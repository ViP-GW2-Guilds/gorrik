package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/ViP-GW2-Guilds/gorrik/agent/api"
	"github.com/ViP-GW2-Guilds/gorrik/agent/parser"
	"github.com/spf13/cobra"
)

var (
	reparseDir     string
	reparseDryRun  bool
	reparseWorkers int
)

var reparseCmd = &cobra.Command{
	Use:   "reparse",
	Short: "Re-parse local logs and update existing database records in place",
	Long: `Walks the log directory, re-parses every .evtc / .zevtc file, and updates
the matching database record's parser-derived fields (result, mode, duration,
encounter). dps.report URLs, favourites and tags are preserved, and logs that
are not in the database are left alone — nothing is uploaded or deleted.

Run this after a parser fix to correct already-imported records. Use --dry-run
first to see how many results would change.

Only files present in the directory are re-parsed, so run it once per directory
if your logs are split across several.`,
	Args: cobra.NoArgs,
	RunE: runReparse,
}

func init() {
	reparseCmd.Flags().StringVar(&reparseDir, "dir", "", "directory to re-parse (defaults to arcdps log directory from config)")
	reparseCmd.Flags().BoolVar(&reparseDryRun, "dry-run", false, "report what would change without writing")
	reparseCmd.Flags().IntVar(&reparseWorkers, "workers", 8, "number of parallel workers")
}

func runReparse(cmd *cobra.Command, args []string) error {
	if cfg.API.URL == "" {
		return fmt.Errorf("api.url is not configured — run 'gorrik setup'")
	}

	dir := reparseDir
	if dir == "" {
		var err error
		if dir, err = resolveLogDir(cfg); err != nil {
			return err
		}
	} else if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("directory %s: %w", dir, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		log.Println("reparse interrupted")
		cancel()
	}()

	var files []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && isLogFile(path) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk %s: %w", dir, err)
	}
	log.Printf("found %d log files in %s", len(files), dir)

	client := api.New(cfg, false)

	var changed, unchanged, notInDB, failed atomic.Int64

	queue := make(chan string, reparseWorkers*2)
	var wg sync.WaitGroup
	for i := 0; i < reparseWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range queue {
				if ctx.Err() != nil {
					return
				}
				base := filepath.Base(path)

				meta, err := parser.Parse(path)
				if err != nil {
					log.Printf("skip %s: parse: %v", base, err)
					failed.Add(1)
					continue
				}

				res, err := client.ReparseLog(ctx, path, meta, reparseDryRun)
				if err != nil {
					log.Printf("failed %s: %v", base, err)
					failed.Add(1)
					continue
				}

				switch {
				case res.Status == "not_found":
					notInDB.Add(1)
				case res.ResultChanged:
					verb := "changed"
					if reparseDryRun {
						verb = "would change"
					}
					log.Printf("%s %s: %s → %s (%s)", verb, base, res.OldResult, res.NewResult, meta.EncounterName)
					changed.Add(1)
				default:
					unchanged.Add(1)
				}
			}
		}()
	}
	for _, f := range files {
		select {
		case queue <- f:
		case <-ctx.Done():
		}
	}
	close(queue)
	wg.Wait()

	action := "result changed"
	if reparseDryRun {
		action = "results would change"
	}
	log.Printf("done: %d %s, %d unchanged, %d not in DB, %d parse failures",
		changed.Load(), action, unchanged.Load(), notInDB.Load(), failed.Load())
	if reparseDryRun && changed.Load() > 0 {
		log.Println("run without --dry-run to apply")
	}
	return nil
}
