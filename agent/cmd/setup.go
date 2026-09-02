package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/ViP-GW2-Guilds/gorrik/agent/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive first-run configuration wizard",
	Long: `Walks through all required settings and writes gorrik.toml.
Safe to re-run at any time to update your configuration; press Enter to
keep the current value shown in [brackets].`,
	Args: cobra.NoArgs,
	RunE: runSetup,
}

func runSetup(cmd *cobra.Command, args []string) error {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("gorrik setup needs an interactive terminal; edit %s directly instead", cfgPath)
	}

	in := bufio.NewReader(os.Stdin)
	c := cfg // pre-populated from existing config or defaults

	// ── arcdps log directory ──────────────────────────────────────────────────
	section("arcdps log directory")
	fmt.Println("Gorrik needs to know where arcdps saves your combat logs.")

	logDir := c.Arcdps.LogDir
	if detected := config.DetectArcdpsLogDir(); detected != "" && logDir == "" {
		logDir = detected
	}
	autoDetect := askBool(in, "Auto-detect the log directory from arcdps.ini?", c.Arcdps.AutoDetect)
	if !autoDetect {
		logDir = ask(in, "Log directory path", logDir)
	}

	// ── Web API ───────────────────────────────────────────────────────────────
	section("Web API")
	fmt.Println("Your Gorrik web app URL and API key for writing logs.")
	apiURL := ask(in, "API URL", c.API.URL)
	apiKey := askSecret("API key", c.API.Key)

	// ── Cloudflare R2 storage ─────────────────────────────────────────────────
	section("Cloudflare R2 storage")
	fmt.Println("Raw .evtc files are archived to your R2 bucket. Credentials are in")
	fmt.Println("the Cloudflare dashboard under R2 → Manage API Tokens.")
	r2AccountID := ask(in, "R2 Account ID", c.Storage.R2AccountID)
	r2AccessKeyID := ask(in, "R2 Access Key ID", c.Storage.R2AccessKeyID)
	r2SecretKey := askSecret("R2 Secret Access Key", c.Storage.R2SecretAccessKey)
	r2Bucket := c.Storage.R2Bucket
	if r2Bucket == "" {
		r2Bucket = "gorrik-logs"
	}
	r2Bucket = ask(in, "R2 Bucket name", r2Bucket)

	// ── Behaviour ─────────────────────────────────────────────────────────────
	section("Behaviour")
	deleteAfter := askBool(in, "Delete local files after confirmed upload? (originals stay in R2)", c.Behaviour.DeleteAfterUpload)

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

	fmt.Printf("\nConfiguration saved to %s\n", cfgPath)
	fmt.Println("Run 'gorrik watch' to start watching for new logs.")
	return nil
}

func section(title string) {
	fmt.Printf("\n── %s ──\n", title)
}

// ask prompts for a line of input, returning current if the user just presses Enter.
func ask(in *bufio.Reader, label, current string) string {
	if current != "" {
		fmt.Printf("%s [%s]: ", label, current)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, _ := in.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return current
	}
	return line
}

// askSecret prompts without echoing input. An empty response keeps the current value.
func askSecret(label, current string) string {
	if current != "" {
		fmt.Printf("%s [keep current]: ", label)
	} else {
		fmt.Printf("%s: ", label)
	}
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil || len(strings.TrimSpace(string(b))) == 0 {
		return current
	}
	return strings.TrimSpace(string(b))
}

// askBool prompts for a yes/no answer, defaulting to current.
func askBool(in *bufio.Reader, label string, current bool) bool {
	def := "y/N"
	if current {
		def = "Y/n"
	}
	fmt.Printf("%s [%s]: ", label, def)
	line, _ := in.ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "":
		return current
	case "y", "yes":
		return true
	default:
		return false
	}
}
