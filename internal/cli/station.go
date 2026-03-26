// Package cli implements station management commands.
package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/uttufy/FactoryAI/internal/store"
)

// stationCmd represents the station command
var stationCmd = &cobra.Command{
	Use:   "station",
	Short: "Manage factory stations",
	Long:  `Manage factory stations - isolated git worktrees where operators work.`,
}

// stationAddCmd represents the station add command
var stationAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new station",
	Long: `Add a new station to the factory.

This creates:
- A git worktree for isolation
- A tmux pane for the operator`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		if err := addStation(name); err != nil {
			exitWithError("Failed to add station", err)
		}
	},
}

// stationListCmd represents the station list command
var stationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all stations",
	Long:  `List all stations in the factory.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := listStations(); err != nil {
			exitWithError("Failed to list stations", err)
		}
	},
}

// stationRemoveCmd represents the station remove command
var stationRemoveCmd = &cobra.Command{
	Use:   "remove <id|name>",
	Short: "Remove a station",
	Long: `Remove a station from the factory.

This will:
- Remove the git worktree
- Kill the tmux pane
- Clean up any associated resources`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		if err := removeStation(id); err != nil {
			exitWithError("Failed to remove station", err)
		}
	},
}

// stationStatusCmd represents the station status command
var stationStatusCmd = &cobra.Command{
	Use:   "status <id|name>",
	Short: "Show station status",
	Long:  `Show detailed status of a specific station.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		if err := showStationStatus(id); err != nil {
			exitWithError("Failed to get station status", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(stationCmd)
	stationCmd.AddCommand(stationAddCmd)
	stationCmd.AddCommand(stationListCmd)
	stationCmd.AddCommand(stationRemoveCmd)
	stationCmd.AddCommand(stationStatusCmd)
}

func addStation(name string) error {
	factory, err := getOrCreateFactory()
	if err != nil {
		return err
	}

	ctx := context.Background()

	station, err := factory.StationManager.Provision(ctx, name)
	if err != nil {
		return err
	}

	fmt.Printf("Station created successfully!\n")
	fmt.Printf("  ID: %s\n", station.ID)
	fmt.Printf("  Name: %s\n", station.Name)
	fmt.Printf("  Worktree: %s\n", station.WorktreePath)
	fmt.Printf("  Status: %s\n", station.Status)

	return nil
}

func listStations() error {
	absPath, err := getAbsPath()
	if err != nil {
		return err
	}

	cfg, err := loadConfig(absPath)
	if err != nil {
		return err
	}

	dbPath := fmt.Sprintf("%s/%s", absPath, cfg.Database.Path)
	s, err := store.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer s.Close()

	ctx := context.Background()
	stations, err := s.ListStations(ctx)
	if err != nil {
		return fmt.Errorf("listing stations: %w", err)
	}

	if len(stations) == 0 {
		fmt.Println("No stations found.")
		fmt.Println("\nTo add a station, run: factory station add <name>")
		return nil
	}

	fmt.Println("STATIONS")
	fmt.Println("========")
	fmt.Printf("%-20s %-10s %-15s %s\n", "NAME", "STATUS", "CURRENT JOB", "WORKTREE")
	fmt.Println("------------------------------------------------------------------------")

	for _, station := range stations {
		job := station.CurrentJob
		if job == "" {
			job = "-"
		}
		fmt.Printf("%-20s %-10s %-15s %s\n",
			station.Name,
			station.Status,
			job,
			station.WorktreePath,
		)
	}

	return nil
}

func removeStation(id string) error {
	factory, err := getOrCreateFactory()
	if err != nil {
		return err
	}

	ctx := context.Background()

	if err := factory.StationManager.Decommission(ctx, id); err != nil {
		return err
	}

	fmt.Printf("Station '%s' removed successfully.\n", id)
	return nil
}

func showStationStatus(id string) error {
	factory, err := getOrCreateFactory()
	if err != nil {
		return err
	}

	ctx := context.Background()

	station, err := factory.StationManager.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("getting station: %w", err)
	}

	fmt.Println("STATION STATUS")
	fmt.Println("==============")
	fmt.Printf("  ID: %s\n", station.ID)
	fmt.Printf("  Name: %s\n", station.Name)
	fmt.Printf("  Status: %s\n", station.Status)
	fmt.Printf("  Worktree: %s\n", station.WorktreePath)
	fmt.Printf("  Tmux Session: %s\n", station.TmuxSession)
	fmt.Printf("  Tmux Window: %d\n", station.TmuxWindow)
	fmt.Printf("  Tmux Pane: %d\n", station.TmuxPane)

	if station.CurrentJob != "" {
		fmt.Printf("  Current Job: %s\n", station.CurrentJob)
	}

	if station.OperatorID != "" {
		fmt.Printf("  Operator: %s\n", station.OperatorID)
	}

	fmt.Printf("  Created: %s\n", station.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Last Activity: %s\n", station.LastActivity.Format("2006-01-02 15:04:05"))

	return nil
}

func getAbsPath() (string, error) {
	if projectPath == "." {
		return os.Getwd()
	}
	return projectPath, nil
}
