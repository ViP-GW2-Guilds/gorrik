package cmd

import (
	"fmt"
	"os"

	"github.com/kardianos/service"
	"github.com/spf13/cobra"
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage the Gorrik Windows service",
}

var serviceInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Gorrik as a Windows service",
	Long: `Registers Gorrik as a Windows service that runs gorrik watch automatically
on login, without requiring a terminal window to remain open.

Requires administrator privileges on Windows.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return manageService("install")
	},
}

var serviceUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove the Gorrik Windows service",
	RunE: func(cmd *cobra.Command, args []string) error {
		return manageService("uninstall")
	},
}

var serviceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Gorrik service",
	RunE: func(cmd *cobra.Command, args []string) error {
		return manageService("start")
	},
}

var serviceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the Gorrik service",
	RunE: func(cmd *cobra.Command, args []string) error {
		return manageService("stop")
	},
}

func init() {
	serviceCmd.AddCommand(serviceInstallCmd)
	serviceCmd.AddCommand(serviceUninstallCmd)
	serviceCmd.AddCommand(serviceStartCmd)
	serviceCmd.AddCommand(serviceStopCmd)
}

func manageService(action string) error {
	svcConfig := serviceConfig()

	// Dummy program for service management operations (no actual run needed).
	svc, err := service.New(&noopProgram{}, svcConfig)
	if err != nil {
		return fmt.Errorf("create service handle: %w", err)
	}

	switch action {
	case "install":
		if err := svc.Install(); err != nil {
			return fmt.Errorf("install service: %w", err)
		}
		fmt.Fprintln(os.Stdout, "Service installed. Run 'gorrik service start' to start it.")
	case "uninstall":
		if err := svc.Uninstall(); err != nil {
			return fmt.Errorf("uninstall service: %w", err)
		}
		fmt.Fprintln(os.Stdout, "Service uninstalled.")
	case "start":
		if err := svc.Start(); err != nil {
			return fmt.Errorf("start service: %w", err)
		}
		fmt.Fprintln(os.Stdout, "Service started.")
	case "stop":
		if err := svc.Stop(); err != nil {
			return fmt.Errorf("stop service: %w", err)
		}
		fmt.Fprintln(os.Stdout, "Service stopped.")
	}
	return nil
}

// serviceConfig returns the shared kardianos/service configuration.
// The service runs 'gorrik watch' using the same config file as the CLI.
func serviceConfig() *service.Config {
	executable, _ := os.Executable()
	return &service.Config{
		Name:        "GorrikAgent",
		DisplayName: "Gorrik Log Agent",
		Description: "Watches your arcdps log directory and archives logs to the cloud.",
		Arguments:   []string{"watch", "--config", cfgPath},
		Executable:  executable,
	}
}

type noopProgram struct{}

func (p *noopProgram) Start(s service.Service) error { return nil }
func (p *noopProgram) Stop(s service.Service) error  { return nil }
