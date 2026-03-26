package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/uttufy/FactoryAI/internal/beads"
)

// getJobCmd returns the job parent command with all subcommands
func getJobCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "job",
		Short: "Job management commands",
	}
	cmd.AddCommand(jobCreateCmd, jobListCmd, jobShowCmd, jobCloseCmd)
	return cmd
}

var jobCreateCmd = &cobra.Command{
	Use:   "create <title>",
	Short: "Create a job ticket (bead)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if beadsClient == nil {
			return fmt.Errorf("factory not initialized")
		}

		title := args[0]
		bead, err := beadsClient.Create("job_ticket", title)
		if err != nil {
			return err
		}

		printSuccess("Job '%s' created with ID: %s", title, bead.ID)
		return nil
	},
}

var jobListCmd = &cobra.Command{
	Use:   "list",
	Short: "List job tickets",
	RunE: func(cmd *cobra.Command, args []string) error {
		if beadsClient == nil {
			return fmt.Errorf("factory not initialized")
		}

		beads, err := beadsClient.List(beads.BeadFilter{Type: beads.BeadJobTicket})
		if err != nil {
			return err
		}

		if len(beads) == 0 {
			printInfo("No job tickets found")
			return nil
		}

		fmt.Println("Job Tickets:")
		for _, b := range beads {
			fmt.Printf("  %s: %s (%s)\n", b.ID, b.Title, b.Status)
		}
		return nil
	},
}

var jobShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show ticket details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if beadsClient == nil {
			return fmt.Errorf("factory not initialized")
		}

		id := args[0]
		bead, err := beadsClient.Get(id)
		if err != nil {
			return err
		}

		fmt.Printf("ID: %s\n", bead.ID)
		fmt.Printf("Title: %s\n", bead.Title)
		fmt.Printf("Status: %s\n", bead.Status)
		if bead.Description != "" {
			fmt.Printf("Description: %s\n", bead.Description)
		}
		if bead.Assignee != "" {
			fmt.Printf("Assignee: %s\n", bead.Assignee)
		}
		fmt.Printf("Created: %s\n", bead.CreatedAt)
		return nil
	},
}

var jobCloseCmd = &cobra.Command{
	Use:   "close <id>",
	Short: "Close a ticket",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if beadsClient == nil {
			return fmt.Errorf("factory not initialized")
		}

		id := args[0]
		if err := beadsClient.Close(id); err != nil {
			return err
		}

		printSuccess("Job '%s' closed", id)
		return nil
	},
}
