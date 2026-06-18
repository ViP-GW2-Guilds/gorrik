package watcher

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/ViP-GW2-Guilds/gorrik/agent/api"
	"github.com/ViP-GW2-Guilds/gorrik/agent/config"
	"github.com/ViP-GW2-Guilds/gorrik/agent/dpsreport"
	"github.com/ViP-GW2-Guilds/gorrik/agent/parser"
	"github.com/ViP-GW2-Guilds/gorrik/agent/uploader"
)

// Watcher monitors a directory for new EVTC log files and processes them.
type Watcher struct {
	cfg       *config.Config
	up        *uploader.Uploader
	client    *api.Client
	dpsClient *dpsreport.Client // nil disables dps.report upload
	dryRun    bool
	done      chan struct{}
	once      sync.Once
}

// New creates a Watcher. dpsClient may be nil to disable dps.report uploads.
func New(cfg *config.Config, up *uploader.Uploader, client *api.Client, dpsClient *dpsreport.Client, dryRun bool) *Watcher {
	return &Watcher{
		cfg:       cfg,
		up:        up,
		client:    client,
		dpsClient: dpsClient,
		dryRun:    dryRun,
		done:      make(chan struct{}),
	}
}

// Run watches the configured log directory until ctx is cancelled.
func (w *Watcher) Run(ctx context.Context) error {
	logDir, err := w.resolveLogDir()
	if err != nil {
		return err
	}

	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create file watcher: %w", err)
	}
	defer fw.Close()

	if err := fw.Add(logDir); err != nil {
		return fmt.Errorf("watch %s: %w", logDir, err)
	}

	log.Printf("watching %s for new logs", logDir)

	for {
		select {
		case <-ctx.Done():
			return nil

		case event, ok := <-fw.Events:
			if !ok {
				return nil
			}
			if event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) {
				if isLogFile(event.Name) {
					go w.processFile(ctx, event.Name)
				}
			}

		case err, ok := <-fw.Errors:
			if !ok {
				return nil
			}
			log.Printf("watcher error: %v", err)
		}
	}
}

// processFile parses, uploads, and posts a single log file.
func (w *Watcher) processFile(ctx context.Context, path string) {
	// Brief pause: arcdps writes atomically via rename, but we wait a moment
	// in case the file is still being finalised by the OS.
	time.Sleep(500 * time.Millisecond)

	if err := w.process(ctx, path); err != nil {
		log.Printf("error processing %s: %v", filepath.Base(path), err)
	}
}

// process runs the full pipeline (parse → R2 upload → dps.report upload → API POST) for one file.
func (w *Watcher) process(ctx context.Context, path string) error {
	log.Printf("processing %s", filepath.Base(path))

	meta, err := parser.Parse(path)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	key, err := w.up.Upload(ctx, path)
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}

	fileURL := uploader.FileURL(w.cfg.Storage.R2AccountID, w.cfg.Storage.R2Bucket, key)
	if w.dryRun {
		fileURL = "dry-run://" + key
	}

	var dpsURL string
	if w.dpsClient != nil && !w.dryRun {
		permalink, err := w.dpsClient.Upload(ctx, path)
		if err != nil {
			log.Printf("warning: dps.report upload failed for %s: %v", filepath.Base(path), err)
		} else {
			dpsURL = permalink
			log.Printf("dps.report: %s → %s", filepath.Base(path), permalink)
		}
	}

	if err := w.client.PostLog(ctx, path, meta, fileURL, dpsURL); err != nil {
		return fmt.Errorf("api post: %w", err)
	}

	log.Printf("done: %s → %s / %s / %s", filepath.Base(path), meta.EncounterName, meta.Result, meta.Mode)

	if w.cfg.Behaviour.DeleteAfterUpload && !w.dryRun {
		if err := os.Remove(path); err != nil {
			log.Printf("warning: could not delete %s: %v", path, err)
		}
	}

	return nil
}

// resolveLogDir returns the log directory from config or arcdps auto-detection.
func (w *Watcher) resolveLogDir() (string, error) {
	if w.cfg.Arcdps.LogDir != "" {
		return expandTilde(w.cfg.Arcdps.LogDir), nil
	}
	if w.cfg.Arcdps.AutoDetect {
		if dir := config.DetectArcdpsLogDir(); dir != "" {
			return dir, nil
		}
	}
	return "", fmt.Errorf("could not determine log directory — set arcdps.log_dir in config or run 'gorrik setup'")
}

// expandTilde replaces a leading ~ with the current user's home directory.
func expandTilde(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[1:])
}

func isLogFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".evtc") ||
		strings.HasSuffix(lower, ".evtc.zip") ||
		strings.HasSuffix(lower, ".zevtc")
}

// BulkImport processes all existing log files in dir using up to workers
// goroutines. dps.report uploads are intentionally skipped — use gorrik backfill-dps
// for historical logs.
func BulkImport(ctx context.Context, dir string, workers int, cfg *config.Config, up *uploader.Uploader, client *api.Client, dryRun bool) error {
	w := New(cfg, up, client, nil, dryRun)

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

	queue := make(chan string, workers*2)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range queue {
				if ctx.Err() != nil {
					return
				}
				if err := w.process(ctx, path); err != nil {
					log.Printf("skip %s: %v", filepath.Base(path), err)
				}
			}
		}()
	}

	for _, f := range files {
		select {
		case queue <- f:
		case <-ctx.Done():
			break
		}
	}
	close(queue)
	wg.Wait()

	return nil
}
