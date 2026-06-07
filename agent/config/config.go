package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Arcdps    ArcdpsConfig    `toml:"arcdps"`
	API       APIConfig       `toml:"api"`
	Storage   StorageConfig   `toml:"storage"`
	Behaviour BehaviourConfig `toml:"behaviour"`
}

type ArcdpsConfig struct {
	AutoDetect bool   `toml:"auto_detect"`
	LogDir     string `toml:"log_dir"`
}

type APIConfig struct {
	URL string `toml:"url"`
	Key string `toml:"key"`
}

type StorageConfig struct {
	R2AccountID       string `toml:"r2_account_id"`
	R2AccessKeyID     string `toml:"r2_access_key_id"`
	R2SecretAccessKey string `toml:"r2_secret_access_key"`
	R2Bucket          string `toml:"r2_bucket"`
}

type BehaviourConfig struct {
	DeleteAfterUpload bool `toml:"delete_after_upload"`
}

// Defaults returns a Config with sensible defaults.
func Defaults() *Config {
	return &Config{
		Arcdps: ArcdpsConfig{
			AutoDetect: true,
		},
		Behaviour: BehaviourConfig{
			DeleteAfterUpload: false,
		},
	}
}

// Load reads gorrik.toml from path. If the file does not exist, defaults are
// returned without error so callers can proceed to the setup wizard.
func Load(path string) (*Config, error) {
	cfg := Defaults()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return cfg, nil
}

// Save writes cfg to path in TOML format.
func Save(cfg *Config, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}

// DefaultPath returns the platform-appropriate default config file location.
// On Windows this is %APPDATA%\gorrik\gorrik.toml; elsewhere it is
// ~/.config/gorrik/gorrik.toml.
func DefaultPath() string {
	if runtime.GOOS == "windows" {
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			return filepath.Join(appdata, "gorrik", "gorrik.toml")
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", "gorrik", "gorrik.toml")
	}
	return "gorrik.toml"
}

// DetectArcdpsLogDir reads arcdps.ini from the default Windows location and
// returns the configured log directory. Returns an empty string if the file
// cannot be found or parsed.
func DetectArcdpsLogDir() string {
	userProfile := os.Getenv("USERPROFILE")
	if userProfile == "" {
		return ""
	}
	iniPath := filepath.Join(userProfile, "Documents", "Guild Wars 2", "addons", "arcdps", "arcdps.ini")
	data, err := os.ReadFile(iniPath)
	if err != nil {
		return ""
	}
	// Simple INI parser: look for "boss_encounter_savepath=<value>" under [session].
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "boss_encounter_savepath=") {
			return strings.TrimPrefix(line, "boss_encounter_savepath=")
		}
	}
	return ""
}

// Validate returns an error if cfg is missing required fields for normal operation.
// api.url and api.key are not required here — a missing URL is handled gracefully
// by the API client (it skips the POST with a warning until Phase 3 is deployed).
func Validate(cfg *Config) error {
	if cfg.Storage.R2AccountID == "" {
		return fmt.Errorf("storage.r2_account_id is not set — run 'gorrik setup' to configure")
	}
	if cfg.Storage.R2Bucket == "" {
		return fmt.Errorf("storage.r2_bucket is not set — run 'gorrik setup' to configure")
	}
	if cfg.Arcdps.LogDir == "" && !cfg.Arcdps.AutoDetect {
		return fmt.Errorf("arcdps.log_dir is not set and auto_detect is false — run 'gorrik setup' to configure")
	}
	return nil
}
