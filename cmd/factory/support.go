package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// getSupportCmds returns support service commands
func getSupportCmds() []*cobra.Command {
	return []*cobra.Command{
		supportStatusCmd,
		supportLogsCmd,
		supportAttachCmd,
	}
}

var supportStatusCmd = &cobra.Command{
	Use:   "support status",
	Short: "Show support services status",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		// TODO: Show support services status
		fmt.Println("Support Services:")
		fmt.Println("  Logger: Running")
		fmt.Println("  Monitor: Running")
		fmt.Println("  Notifier: Running")
		return nil
	},
}

var supportLogsCmd = &cobra.Command{
	Use:   "support logs [service]",
	Short: "Show support service logs",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		service := "all"
		if len(args) > 0 {
			service = args[0]
		}
		// TODO: Show logs
		printInfo("Logs for service: %s", service)
		return nil
	},
}

var supportAttachCmd = &cobra.Command{
	Use:   "support attach <station>",
	Short: "Attach support to station",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		stationID := args[0]
		// TODO: Attach support to station
		printSuccess("Support attached to station %s", stationID)
		return nil
	},
}
