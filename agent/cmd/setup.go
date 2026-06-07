package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/ViP-GW2-Guilds/gorrik/agent/config"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive first-run configuration wizard",
	Long: `Walks through all required settings and writes gorrik.toml.
Safe to re-run at any time to update your configuration.`,
	RunE: runSetup,
}

func runSetup(cmd *cobra.Command, args []string) error {
	// Pre-populate from existing config (or defaults) so re-runs are non-destructive.
	c := cfg

	// ── arcdps log directory ──────────────────────────────────────────────────

	detectedDir := config.DetectArcdpsLogDir()
	autoDetect := c.Arcdps.AutoDetect
	logDir := c.Arcdps.LogDir

	if detectedDir != "" && logDir == "" {
		logDir = detectedDir
	}

	arcdpsForm := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("arcdps log directory").
				Description("Gorrik needs to know where arcdps saves your combat logs."),
			huh.NewConfirm().
				Title("Auto-detect from arcdps.ini?").
				Value(&autoDetect),
			huh.NewInput().
				Title("Log directory path").
				Description("Only used if auto-detect is off or fails.").
				Placeholder(`C:\Users\You\Documents\Guild Wars 2\addons\arcdps\arcdps.local`).
				Value(&logDir),
		),
	)

	if err := arcdpsForm.Run(); err != nil {
		return fmt.Errorf("setup cancelled: %w", err)
	}

	// ── API ───────────────────────────────────────────────────────────────────

	apiURL := c.API.URL
	apiKey := c.API.Key

	apiForm := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Web API").
				Description("Your Gorrik web app URL and API key for writing logs."),
			huh.NewInput().
				Title("API URL").
				Placeholder("https://gorrik.vercel.app/api").
				Value(&apiURL),
			huh.NewInput().
				Title("API key").
				EchoMode(huh.EchoModePassword).
				Value(&apiKey),
		),
	)

	if err := apiForm.Run(); err != nil {
		return fmt.Errorf("setup cancelled: %w", err)
	}

	// ── R2 storage ────────────────────────────────────────────────────────────

	r2AccountID := c.Storage.R2AccountID
	r2AccessKeyID := c.Storage.R2AccessKeyID
	r2SecretKey := c.Storage.R2SecretAccessKey
	r2Bucket := c.Storage.R2Bucket

	if r2Bucket == "" {
		r2Bucket = "gorrik-logs"
	}

	r2Form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Cloudflare R2 storage").
				Description("Raw .evtc files are archived to your R2 bucket.\nFind these in the Cloudflare dashboard under R2 → Manage API Tokens."),
			huh.NewInput().
				Title("R2 Account ID").
				Value(&r2AccountID),
			huh.NewInput().
				Title("R2 Access Key ID").
				Value(&r2AccessKeyID),
			huh.NewInput().
				Title("R2 Secret Access Key").
				EchoMode(huh.EchoModePassword).
				Value(&r2SecretKey),
			huh.NewInput().
				Title("R2 Bucket name").
				Value(&r2Bucket),
		),
	)

	if err := r2Form.Run(); err != nil {
		return fmt.Errorf("setup cancelled: %w", err)
	}

	// ── Behaviour ─────────────────────────────────────────────────────────────

	deleteAfter := c.Behaviour.DeleteAfterUpload

	behaviourForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Delete local files after confirmed upload?").
				Description("Keeps your disk tidy. Originals remain in R2.").
				Value(&deleteAfter),
		),
	)

	if err := behaviourForm.Run(); err != nil {
		return fmt.Errorf("setup cancelled: %w", err)
	}

	// ── Write config ──────────────────────────────────────────────────────────

	c.Arcdps.AutoDetect = autoDetect
	c.Arcdps.LogDir = logDir
	c.API.URL = apiURL
	c.API.Key = apiKey
	c.Storage.R2AccountID = r2AccountID
	c.Storage.R2AccessKeyID = r2AccessKeyID
	c.Storage.R2SecretAccessKey = r2SecretKey
	c.Storage.R2Bucket = r2Bucket
	c.Behaviour.DeleteAfterUpload = deleteAfter

	if err := config.Save(c, cfgPath); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(os.Stdout, "\nConfiguration saved to %s\n", cfgPath)
	fmt.Fprintln(os.Stdout, "Run 'gorrik watch' to start watching for new logs.")
	return nil
}
