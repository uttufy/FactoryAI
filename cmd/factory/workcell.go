package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// getWorkCellCmds returns work cell management commands
func getWorkCellCmds() []*cobra.Command {
	return []*cobra.Command{
		cellCreateCmd,
		cellActivateCmd,
		cellStatusCmd,
		cellDisperseCmd,
	}
}

var cellCreateCmd = &cobra.Command{
	Use:   "cell create <name> <stations...>",
	Short: "Create a work cell",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		ctx := context.Background()
		name := args[0]
		stations := args[1:]
		// For now, create with empty bead IDs (can be assigned later)
		cell, err := workCellManager.Create(ctx, name, stations, []string{})
		if err != nil {
			return err
		}

		printSuccess("Work cell '%s' created with %d stations", cell.ID, len(stations))
		return nil
	},
}

var cellActivateCmd = &cobra.Command{
	Use:   "cell activate <cell-id>",
	Short: "Activate parallel execution",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		ctx := context.Background()
		cellID := args[0]
		if err := workCellManager.Activate(ctx, cellID); err != nil {
			return err
		}

		printSuccess("Work cell '%s' activated", cellID)
		return nil
	},
}

var cellStatusCmd = &cobra.Command{
	Use:   "cell status <cell-id>",
	Short: "Show cell status",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		ctx := context.Background()
		cellID := args[0]
		cell, err := workCellManager.Status(ctx, cellID)
		if err != nil {
			return err
		}

		fmt.Printf("Work Cell: %s\n", cell.Name)
		fmt.Printf("ID: %s\n", cell.ID)
		fmt.Printf("Status: %s\n", cell.Status)
		fmt.Printf("Stations: %v\n", cell.Stations)
		fmt.Printf("Created: %s\n", cell.CreatedAt.Format("2006-01-02 15:04:05"))
		return nil
	},
}

var cellDisperseCmd = &cobra.Command{
	Use:   "cell disperse <cell-id>",
	Short: "Disperse cell",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		ctx := context.Background()
		cellID := args[0]
		if err := workCellManager.Disperse(ctx, cellID); err != nil {
			return err
		}

		printSuccess("Work cell '%s' dispersed", cellID)
		return nil
	},
}
