package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// getStationCmd returns the station parent command with all subcommands
func getStationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "station",
		Short: "Station management commands",
	}
	cmd.AddCommand(stationAddCmd, stationListCmd, stationRemoveCmd, stationStatusCmd)
	return cmd
}

var stationAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Provision a new station",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		ctx := context.Background()
		name := args[0]
		station, err := stationManager.Provision(ctx, name)
		if err != nil {
			return err
		}

		printSuccess("Station '%s' provisioned at %s", station.Name, station.WorktreePath)
		return nil
	},
}

var stationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all stations",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		ctx := context.Background()
		stations := stationManager.List(ctx)
		if len(stations) == 0 {
			printInfo("No stations found")
			return nil
		}

		fmt.Println("Stations:")
		for _, s := range stations {
			job := s.CurrentJob
			if job == "" {
				job = "idle"
			}
			fmt.Printf("  %s (%s): %s - %s\n", s.ID, s.Name, s.Status, job)
		}
		return nil
	},
}

var stationRemoveCmd = &cobra.Command{
	Use:   "remove <id>",
	Short: "Decommission a station",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		ctx := context.Background()
		id := args[0]
		if err := stationManager.Decommission(ctx, id); err != nil {
			return err
		}

		printSuccess("Station '%s' decommissioned", id)
		return nil
	},
}

var stationStatusCmd = &cobra.Command{
	Use:   "status <id>",
	Short: "Show station status",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		ctx := context.Background()
		id := args[0]
		station, err := stationManager.Get(ctx, id)
		if err != nil {
			return err
		}

		fmt.Printf("Station: %s\n", station.Name)
		fmt.Printf("ID: %s\n", station.ID)
		fmt.Printf("Status: %s\n", station.Status)
		fmt.Printf("Worktree: %s\n", station.WorktreePath)
		fmt.Printf("Tmux Session: %s\n", station.TmuxSession)
		if station.CurrentJob != "" {
			fmt.Printf("Current Job: %s\n", station.CurrentJob)
		}
		fmt.Printf("Created: %s\n", station.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Last Activity: %s\n", station.LastActivity.Format("2006-01-02 15:04:05"))
		return nil
	},
}
