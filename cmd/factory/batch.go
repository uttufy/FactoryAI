package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// getBatchCmds returns batch management commands
func getBatchCmds() []*cobra.Command {
	return []*cobra.Command{
		batchCreateCmd,
		batchStatusCmd,
		batchListCmd,
		batchDashboardCmd,
	}
}

var batchCreateCmd = &cobra.Command{
	Use:   "batch create <name> <jobs...>",
	Short: "Create batch",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		name := args[0]
		jobs := args[1:]
		b, err := batchMgr.Create(name, jobs)
		if err != nil {
			return err
		}

		printSuccess("Batch '%s' created with ID: %s", name, b.ID)
		return nil
	},
}

var batchStatusCmd = &cobra.Command{
	Use:   "batch status <id>",
	Short: "Show batch status",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		ctx := context.Background()
		id := args[0]
		batch, err := batchMgr.Track(ctx, id)
		if err != nil {
			return err
		}

		fmt.Printf("Batch: %s\n", batch.Name)
		fmt.Printf("ID: %s\n", batch.ID)
		fmt.Printf("Status: %s\n", batch.Status)
		fmt.Printf("Jobs: %v\n", batch.TrackedIDs)
		fmt.Printf("Created: %s\n", batch.CreatedAt.Format("2006-01-02 15:04:05"))
		return nil
	},
}

var batchListCmd = &cobra.Command{
	Use:   "batch list",
	Short: "List batches",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		ctx := context.Background()
		batches, err := batchMgr.List(ctx, "")
		if err != nil {
			return err
		}

		if len(batches) == 0 {
			printInfo("No batches found")
			return nil
		}

		fmt.Println("Batches:")
		for _, b := range batches {
			fmt.Printf("  %s: %s (%s) - %d jobs\n", b.ID, b.Name, b.Status, len(b.TrackedIDs))
		}
		return nil
	},
}

var batchDashboardCmd = &cobra.Command{
	Use:   "batch dashboard",
	Short: "Show batch dashboard (TUI)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		// TODO: Implement TUI dashboard
		printInfo("TUI dashboard not yet implemented")
		return nil
	},
}
