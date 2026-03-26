// Package cli implements batch management commands.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// batchCmd represents the batch command
var batchCmd = &cobra.Command{
	Use:   "batch",
	Short: "Manage production batches",
	Long:  `Manage production batches - groups of related work items.`,
}

// batchCreateCmd represents the batch create command
var batchCreateCmd = &cobra.Command{
	Use:   "create <name> <job-ids...>",
	Short: "Create a new batch",
	Long: `Create a new production batch from existing jobs.

Example:
  factory batch create "Feature Auth" job-123 job-124 job-125`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		jobIDs := args[1:]
		if err := createBatch(name, jobIDs); err != nil {
			exitWithError("Failed to create batch", err)
		}
	},
}

// batchListCmd represents the batch list command
var batchListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all batches",
	Long:  `List all production batches.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := listBatches(); err != nil {
			exitWithError("Failed to list batches", err)
		}
	},
}

// batchStatusCmd represents the batch status command
var batchStatusCmd = &cobra.Command{
	Use:   "status <batch-id>",
	Short: "Show batch status",
	Long:  `Show detailed status of a specific batch.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		if err := showBatchStatus(id); err != nil {
			exitWithError("Failed to show batch status", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(batchCmd)
	batchCmd.AddCommand(batchCreateCmd)
	batchCmd.AddCommand(batchListCmd)
	batchCmd.AddCommand(batchStatusCmd)
}

func createBatch(name string, jobIDs []string) error {
	factory, err := getOrCreateFactory()
	if err != nil {
		return err
	}

	batch, err := factory.BeadsClient.CreateBatch(name, jobIDs)
	if err != nil {
		return fmt.Errorf("creating batch: %w", err)
	}

	fmt.Printf("Batch created successfully!\n")
	fmt.Printf("  ID: %s\n", batch.ID)
	fmt.Printf("  Name: %s\n", batch.Name)
	fmt.Printf("  Jobs: %d\n", len(jobIDs))

	return nil
}

func listBatches() error {
	factory, err := getOrCreateFactory()
	if err != nil {
		return err
	}

	batches, err := factory.BeadsClient.ListBatches()
	if err != nil {
		return fmt.Errorf("listing batches: %w", err)
	}

	if len(batches) == 0 {
		fmt.Println("No batches found.")
		fmt.Println("\nTo create a batch, run: factory batch create <name> <job-ids...>")
		return nil
	}

	fmt.Println("BATCHES")
	fmt.Println("=======")
	fmt.Printf("%-20s %-15s %-10s %-10s %s\n", "ID", "STATUS", "TOTAL", "DONE", "NAME")
	fmt.Println("------------------------------------------------------------------------")

	for _, batch := range batches {
		fmt.Printf("%-20s %-15s %-10d %-10d %s\n",
			batch.ID,
			batch.Status,
			batch.TotalJobs,
			batch.CompletedJobs,
			batch.Name,
		)
	}

	return nil
}

func showBatchStatus(id string) error {
	factory, err := getOrCreateFactory()
	if err != nil {
		return err
	}

	batch, err := factory.BeadsClient.GetBatch(id)
	if err != nil {
		return fmt.Errorf("getting batch: %w", err)
	}

	fmt.Println("BATCH DETAILS")
	fmt.Println("=============")
	fmt.Printf("  ID: %s\n", batch.ID)
	fmt.Printf("  Name: %s\n", batch.Name)
	fmt.Printf("  Status: %s\n", batch.Status)
	fmt.Printf("  Total Jobs: %d\n", batch.TotalJobs)
	fmt.Printf("  Completed: %d\n", batch.CompletedJobs)
	fmt.Printf("  Failed: %d\n", batch.FailedJobs)

	if batch.Description != "" {
		fmt.Printf("  Description: %s\n", batch.Description)
	}

	fmt.Printf("  Created: %s\n", batch.CreatedAt.Format("2006-01-02 15:04:05"))

	if batch.CompletedAt != nil {
		fmt.Printf("  Completed: %s\n", batch.CompletedAt.Format("2006-01-02 15:04:05"))
	}

	if batch.Result != "" {
		fmt.Printf("  Result: %s\n", batch.Result)
	}

	// Show progress bar
	if batch.TotalJobs > 0 {
		progress := float64(batch.CompletedJobs) / float64(batch.TotalJobs)
		barWidth := 40
		filled := int(progress * float64(barWidth))

		fmt.Printf("\n  Progress: [")
		for i := 0; i < barWidth; i++ {
			if i < filled {
				fmt.Print("=")
			} else {
				fmt.Print(" ")
			}
		}
		fmt.Printf("] %.0f%%\n", progress*100)
	}

	return nil
}
