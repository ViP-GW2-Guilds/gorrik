package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/ViP-GW2-Guilds/gorrik/agent/api"
	"github.com/spf13/cobra"
)

var importDpsUrlsCachePath string

var importDpsUrlsCmd = &cobra.Command{
	Use:   "import-dps-urls",
	Short: "Import dps.report URLs from the arcdps Log Manager cache",
	Long: `Parses the arcdps Log Manager cache file (LogDataCache.json) and
updates any matching log in the database with its stored dps.report URL.
No files are uploaded — this only copies URLs that Log Manager already recorded.

Run this before 'gorrik backfill-dps' to avoid re-uploading files that
Log Manager has already uploaded.

Default cache path: %APPDATA%\arcdps Log Manager\LogDataCache.json`,
	Args: cobra.NoArgs,
	RunE: runImportDpsUrls,
}

func init() {
	importDpsUrlsCmd.Flags().StringVar(&importDpsUrlsCachePath, "cache", "", "path to LogDataCache.json (required)")
	importDpsUrlsCmd.MarkFlagRequired("cache")
}

// logManagerCache mirrors the relevant fields of LogDataCache.json.
type logManagerCache struct {
	LogsByFilename map[string]logManagerEntry `json:"LogsByFilename"`
}

type logManagerEntry struct {
	DpsReportEIUpload *dpsReportUpload `json:"DpsReportEIUpload"`
}

type dpsReportUpload struct {
	Url string `json:"Url"`
}

func runImportDpsUrls(cmd *cobra.Command, args []string) error {
	data, err := os.ReadFile(importDpsUrlsCachePath)
	if err != nil {
		return fmt.Errorf("read cache: %w", err)
	}

	// Log Manager writes the file with a UTF-8 BOM, which encoding/json rejects.
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))

	var cache logManagerCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return fmt.Errorf("parse cache: %w", err)
	}

	// Build base filename → dps.report URL map, deduplicating by base filename.
	cacheURLs := make(map[string]string)
	for fullPath, entry := range cache.LogsByFilename {
		if entry.DpsReportEIUpload == nil || entry.DpsReportEIUpload.Url == "" {
			continue
		}
		base := filepath.Base(fullPath)
		cacheURLs[base] = entry.DpsReportEIUpload.Url
	}
	log.Printf("cache: %d log entries with dps.report URLs", len(cacheURLs))
	if len(cacheURLs) == 0 {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		log.Println("interrupted")
		cancel()
	}()

	apiClient := api.New(cfg, false)

	log.Println("fetching logs without dps.report URL from database...")
	entries, err := apiClient.FetchLogsWithoutDpsURL(ctx)
	if err != nil {
		return err
	}
	log.Printf("%d logs in DB need a dps.report URL", len(entries))

	patched, skipped, failed := 0, 0, 0
	for _, entry := range entries {
		if ctx.Err() != nil {
			break
		}
		url, ok := cacheURLs[entry.Filename]
		if !ok {
			skipped++
			continue
		}
		if err := apiClient.PatchLogDpsURL(ctx, entry.ID, url); err != nil {
			log.Printf("PATCH failed for %s: %v", entry.Filename, err)
			failed++
		} else {
			log.Printf("ok: %s → %s", entry.Filename, url)
			patched++
		}
	}

	log.Printf("done: %d patched, %d not in cache, %d failed", patched, skipped, failed)
	return nil
}
