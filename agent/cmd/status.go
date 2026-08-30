package cmd

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/ViP-GW2-Guilds/gorrik/agent/api"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show config, local log directory, and how far behind the database is",
	Long: `Prints a summary of the current setup: the config file in use, the
resolved arcdps log directory and its size, the number of logs indexed in the
database, and how many local logs are not yet indexed.

Running 'gorrik' with no arguments is equivalent to 'gorrik status'.`,
	Args: cobra.NoArgs,
	RunE: runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()

	// ── Config ────────────────────────────────────────────────────────────────
	fmt.Fprintf(out, "Config:     %s\n", cfgPath)
	if _, err := os.Stat(cfgPath); err != nil {
		fmt.Fprintln(out, "            not found — run 'gorrik setup'")
	}

	// ── Local log directory ───────────────────────────────────────────────────
	dir, dirErr := resolveLogDir(cfg)
	var localNames []string
	if dirErr != nil {
		fmt.Fprintf(out, "Log dir:    %v\n", dirErr)
	} else {
		var count int
		var size int64
		_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !isLogFile(p) {
				return nil
			}
			count++
			localNames = append(localNames, filepath.Base(p))
			if info, err := d.Info(); err == nil {
				size += info.Size()
			}
			return nil
		})
		fmt.Fprintf(out, "Log dir:    %s\n", dir)
		fmt.Fprintf(out, "            %d log files, %s on disk\n", count, humanBytes(size))
	}

	// ── Log Manager cache ─────────────────────────────────────────────────────
	if cache := resolveLogManagerCache(cfg); cache != "" {
		fmt.Fprintf(out, "LM cache:   %s\n", cache)
	} else {
		fmt.Fprintln(out, "LM cache:   not found — dps.report URL import unavailable")
	}

	// ── Database ──────────────────────────────────────────────────────────────
	if cfg.API.URL == "" {
		fmt.Fprintln(out, "Database:   api.url not configured")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	client := api.New(cfg, false)

	stats, err := client.FetchStats(ctx)
	if err != nil {
		fmt.Fprintf(out, "Database:   error: %v\n", err)
		return nil
	}
	fmt.Fprintf(out, "Database:   %d logs indexed\n", stats.Total)
	if stats.NewestLoggedAt != nil {
		fmt.Fprintf(out, "            newest: %s\n", stats.NewestLoggedAt.Format("2006-01-02 15:04 MST"))
	}
	fmt.Fprintf(out, "            %d missing a dps.report URL\n", stats.MissingDpsReport)

	// ── Behind by ─────────────────────────────────────────────────────────────
	behind := -1
	if dirErr == nil {
		missing, err := client.FetchMissingFilenames(ctx, localNames)
		if err != nil {
			fmt.Fprintf(out, "Behind by:  error: %v\n", err)
		} else {
			behind = len(missing)
			fmt.Fprintf(out, "Behind by:  %d local logs not yet indexed\n", behind)
		}
	}

	// ── Suggested next step ───────────────────────────────────────────────────
	fmt.Fprintln(out)
	if behind > 0 || stats.MissingDpsReport > 0 {
		fmt.Fprintln(out, "Next: gorrik sync")
	} else if behind == 0 {
		fmt.Fprintln(out, "Up to date.")
	}
	return nil
}
