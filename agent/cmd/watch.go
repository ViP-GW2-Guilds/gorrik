package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kardianos/service"
	"github.com/spf13/cobra"
	"github.com/ViP-GW2-Guilds/gorrik/agent/api"
	"github.com/ViP-GW2-Guilds/gorrik/agent/config"
	"github.com/ViP-GW2-Guilds/gorrik/agent/uploader"
	"github.com/ViP-GW2-Guilds/gorrik/agent/watcher"
)

var watchDryRun bool

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch the log directory and process new logs as they appear",
	Long: `Runs continuously, processing each new .evtc / .zevtc file as arcdps creates it.
When installed as a Windows service (gorrik service install), the service
runs this command automatically on login without a terminal window.`,
	RunE: runWatch,
}

func init() {
	watchCmd.Flags().BoolVar(&watchDryRun, "dry-run", false, "parse and log what would be uploaded without actually uploading")
}

func runWatch(cmd *cobra.Command, args []string) error {
	if err := config.Validate(cfg); err != nil && !watchDryRun {
		return err
	}

	up, err := uploader.New(cfg, watchDryRun)
	if err != nil {
		return fmt.Errorf("uploader: %w", err)
	}

	apiClient := api.New(cfg, watchDryRun)
	w := watcher.New(cfg, up, apiClient, watchDryRun)

	// If running as a Windows service, hand control to the service manager.
	// Otherwise run in the foreground with signal handling.
	if service.Interactive() {
		return runForeground(w)
	}
	return runAsService(w)
}

// runForeground runs the watcher in the foreground, stopping on SIGINT/SIGTERM.
func runForeground(w *watcher.Watcher) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		log.Println("shutting down...")
		cancel()
	}()

	return w.Run(ctx)
}

// serviceProgram adapts Watcher to the kardianos/service interface.
type serviceProgram struct {
	w      *watcher.Watcher
	cancel context.CancelFunc
}

func (p *serviceProgram) Start(s service.Service) error {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	go func() {
		if err := p.w.Run(ctx); err != nil {
			log.Printf("watcher error: %v", err)
		}
	}()
	return nil
}

func (p *serviceProgram) Stop(s service.Service) error {
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

func runAsService(w *watcher.Watcher) error {
	svcConfig := serviceConfig()
	prog := &serviceProgram{w: w}
	svc, err := service.New(prog, svcConfig)
	if err != nil {
		return fmt.Errorf("create service: %w", err)
	}
	logger, err := svc.Logger(nil)
	if err != nil {
		return err
	}
	if err := svc.Run(); err != nil {
		_ = logger.Error(err)
		return err
	}
	return nil
}
