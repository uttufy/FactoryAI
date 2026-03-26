// Package cli implements job management commands.
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/uttufy/FactoryAI/internal/beads"
)

// jobCmd represents the job command
var jobCmd = &cobra.Command{
	Use:   "job",
	Short: "Manage jobs (work items)",
	Long:  `Manage jobs (work items) via the beads CLI.`,
}

// jobCreateCmd represents the job create command
var jobCreateCmd = &cobra.Command{
	Use:   "create <title>",
	Short: "Create a new job",
	Long: `Create a new job (work item).

The job will be queued and can be dispatched to a station.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		title := args[0]
		if err := createJob(title); err != nil {
			exitWithError("Failed to create job", err)
		}
	},
}

// jobListCmd represents the job list command
var jobListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all jobs",
	Long:  `List all jobs in the system.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := listJobs(); err != nil {
			exitWithError("Failed to list jobs", err)
		}
	},
}

// jobShowCmd represents the job show command
var jobShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show job details",
	Long:  `Show detailed information about a specific job.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		if err := showJob(id); err != nil {
			exitWithError("Failed to show job", err)
		}
	},
}

// jobCloseCmd represents the job close command
var jobCloseCmd = &cobra.Command{
	Use:   "close <id>",
	Short: "Close a job",
	Long:  `Close a job (mark as done).`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		if err := closeJob(id); err != nil {
			exitWithError("Failed to close job", err)
		}
	},
}

// jobDispatchCmd represents the job dispatch command
var jobDispatchCmd = &cobra.Command{
	Use:   "dispatch <job-id> <station-id>",
	Short: "Dispatch a job to a station",
	Long:  `Dispatch a specific job to a specific station.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		jobID := args[0]
		stationID := args[1]
		if err := dispatchJob(jobID, stationID); err != nil {
			exitWithError("Failed to dispatch job", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(jobCmd)
	jobCmd.AddCommand(jobCreateCmd)
	jobCmd.AddCommand(jobListCmd)
	jobCmd.AddCommand(jobShowCmd)
	jobCmd.AddCommand(jobCloseCmd)
	jobCmd.AddCommand(jobDispatchCmd)
}

func createJob(title string) error {
	factory := GetFactory()
	if factory == nil {
		return fmt.Errorf("factory not running. Run 'factory boot' first")
	}

	bead, err := factory.BeadsClient.Create("task", title)
	if err != nil {
		return fmt.Errorf("creating bead: %w", err)
	}

	fmt.Printf("Job created successfully!\n")
	fmt.Printf("  ID: %s\n", bead.ID)
	fmt.Printf("  Title: %s\n", bead.Title)
	fmt.Printf("  Status: %s\n", bead.Status)

	return nil
}

func listJobs() error {
	factory := GetFactory()
	if factory == nil {
		return fmt.Errorf("factory not running. Run 'factory boot' first")
	}

	beadList, err := factory.BeadsClient.List(beads.BeadFilter{Type: beads.BeadTask})
	if err != nil {
		return fmt.Errorf("listing beads: %w", err)
	}

	if len(beadList) == 0 {
		fmt.Println("No jobs found.")
		fmt.Println("\nTo create a job, run: factory job create <title>")
		return nil
	}

	fmt.Println("JOBS")
	fmt.Println("====")
	fmt.Printf("%-20s %-15s %-10s %s\n", "ID", "STATUS", "ASSIGNEE", "TITLE")
	fmt.Println("------------------------------------------------------------------------")

	for _, bead := range beadList {
		assignee := bead.Assignee
		if assignee == "" {
			assignee = "-"
		}
		fmt.Printf("%-20s %-15s %-10s %s\n",
			bead.ID,
			bead.Status,
			assignee,
			bead.Title,
		)
	}

	return nil
}

func showJob(id string) error {
	factory := GetFactory()
	if factory == nil {
		return fmt.Errorf("factory not running. Run 'factory boot' first")
	}

	bead, err := factory.BeadsClient.Get(id)
	if err != nil {
		return fmt.Errorf("getting bead: %w", err)
	}

	fmt.Println("JOB DETAILS")
	fmt.Println("===========")
	fmt.Printf("  ID: %s\n", bead.ID)
	fmt.Printf("  Type: %s\n", bead.Type)
	fmt.Printf("  Title: %s\n", bead.Title)
	fmt.Printf("  Status: %s\n", bead.Status)

	if bead.Description != "" {
		fmt.Printf("  Description: %s\n", bead.Description)
	}
	if bead.Assignee != "" {
		fmt.Printf("  Assignee: %s\n", bead.Assignee)
	}
	if bead.StationID != "" {
		fmt.Printf("  Station: %s\n", bead.StationID)
	}
	if len(bead.Dependencies) > 0 {
		fmt.Printf("  Dependencies: %v\n", bead.Dependencies)
	}
	if len(bead.Labels) > 0 {
		fmt.Printf("  Labels: %v\n", bead.Labels)
	}

	fmt.Printf("  Created: %s\n", bead.CreatedAt)
	fmt.Printf("  Updated: %s\n", bead.UpdatedAt)

	if bead.CompletedAt != nil {
		fmt.Printf("  Completed: %s\n", *bead.CompletedAt)
	}

	if bead.Result != "" {
		fmt.Printf("  Result: %s\n", bead.Result)
	}

	if bead.Error != "" {
		fmt.Printf("  Error: %s\n", bead.Error)
	}

	return nil
}

func closeJob(id string) error {
	factory := GetFactory()
	if factory == nil {
		return fmt.Errorf("factory not running. Run 'factory boot' first")
	}

	if err := factory.BeadsClient.Close(id); err != nil {
		return fmt.Errorf("closing bead: %w", err)
	}

	fmt.Printf("Job '%s' closed successfully.\n", id)
	return nil
}

func dispatchJob(jobID, stationID string) error {
	factory := GetFactory()
	if factory == nil {
		return fmt.Errorf("factory not running. Run 'factory boot' first")
	}

	ctx := context.Background()

	if err := factory.Planner.Dispatch(ctx, jobID, stationID); err != nil {
		return err
	}

	fmt.Printf("Job '%s' dispatched to station '%s'.\n", jobID, stationID)
	return nil
}
