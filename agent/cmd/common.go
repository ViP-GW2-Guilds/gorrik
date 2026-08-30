package cmd

import (
	"fmt"
	"os"

	"github.com/ViP-GW2-Guilds/gorrik/agent/config"
)

// resolveLogDir returns the arcdps log directory from config, falling back to
// auto-detection. It errors if no directory can be determined or the resolved
// path does not exist.
func resolveLogDir(cfg *config.Config) (string, error) {
	dir := cfg.Arcdps.LogDir
	if dir == "" && cfg.Arcdps.AutoDetect {
		dir = config.DetectArcdpsLogDir()
	}
	if dir == "" {
		return "", fmt.Errorf("no log directory configured — set arcdps.log_dir or run 'gorrik setup'")
	}
	if _, err := os.Stat(dir); err != nil {
		return "", fmt.Errorf("log directory %s: %w", dir, err)
	}
	return dir, nil
}

// resolveLogManagerCache returns the configured arcdps Log Manager cache path,
// or the auto-detected default. Returns an empty string if neither is available.
func resolveLogManagerCache(cfg *config.Config) string {
	if cfg.ArcdpsLogManager.CachePath != "" {
		return cfg.ArcdpsLogManager.CachePath
	}
	return config.DetectLogManagerCache()
}

// humanBytes formats a byte count as a human-readable string (e.g. "20.4 GB").
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
