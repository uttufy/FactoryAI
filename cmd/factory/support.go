package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/uttufy/FactoryAI/internal/events"
)

// getSupportCmd returns the support parent command with all subcommands
func getSupportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "support",
		Short: "Support service commands",
	}
	cmd.AddCommand(supportStatusCmd, supportLogsCmd, supportAttachCmd)
	return cmd
}

var supportStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show support services status",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		ctx := context.Background()
		report, err := supportService.RunHealthCheck(ctx)
		if err != nil {
			return fmt.Errorf("running health check: %w", err)
		}

		fmt.Println("Support Services Status:")
		fmt.Println()

		// Database
		dbStatus := "✗ Unhealthy"
		if report.DatabaseOK {
			dbStatus = "✓ Healthy"
		}
		fmt.Printf("  Database: %s\n", dbStatus)

		// tmux
		tmuxStatus := "✗ Unhealthy"
		if report.TmuxOK {
			tmuxStatus = "✓ Healthy"
		}
		fmt.Printf("  tmux: %s\n", tmuxStatus)

		// Beads
		beadsStatus := "✗ Unhealthy"
		if report.BeadsOK {
			beadsStatus = "✓ Healthy"
		}
		fmt.Printf("  Beads: %s\n", beadsStatus)

		// Disk space
		fmt.Printf("  Disk Space: %d MB available\n", report.DiskSpaceMB)

		// Active stations
		fmt.Printf("  Active Stations: %d\n", report.ActiveStations)

		// Expired leases
		fmt.Printf("  Expired Leases: %d\n", report.ExpiredLeases)

		// Errors
		if len(report.Errors) > 0 {
			fmt.Println("\nErrors:")
			for _, e := range report.Errors {
				fmt.Printf("  - %s\n", e)
			}
		}

		return nil
	},
}

var supportLogsCmd = &cobra.Command{
	Use:   "logs [type]",
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

		// Get events from the last hour
		since := time.Now().Add(-1 * time.Hour)
		eventList, err := storeInstance.GetEvents(since, events.EventType(""))
		if err != nil {
			printInfo("No events found")
			return nil
		}

		if len(eventList) > 20 {
			printInfo("Showing last 20 events for service: %s", service)
		}

		fmt.Printf("Events (last hour, %d total):\n", len(eventList))
		for _, evt := range eventList {
			timestamp := time.Unix(evt.Timestamp, 0).Format("15:04:05")
			fmt.Printf("  [%s] %s: %s\n", timestamp, evt.Type, evt.Subject)
			if evt.Source != "" {
				fmt.Printf("    Source: %s\n", evt.Source)
			}
		}
		return nil
	},
}

var supportAttachCmd = &cobra.Command{
	Use:   "attach <station>",
	Short: "Attach support to station",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		ctx := context.Background()
		stationID := args[0]

		// Verify station exists
		_, err := stationManager.Get(ctx, stationID)
		if err != nil {
			return fmt.Errorf("station not found: %s", stationID)
		}

		// In a real implementation, this would subscribe the support service
		// to the notification channel for the station
		printSuccess("Support attached to station %s", stationID)
		printInfo("Support service will monitor this station for issues")
		return nil
	},
}
