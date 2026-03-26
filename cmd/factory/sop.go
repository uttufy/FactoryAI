package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// getSOPCmds returns SOP management commands
func getSOPCmds() []*cobra.Command {
	return []*cobra.Command{
		sopListCmd,
		sopShowCmd,
		sopExecuteCmd,
	}
}

var sopListCmd = &cobra.Command{
	Use:   "sop list",
	Short: "List SOPs",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		sops := dagEngine.ListSOPs()
		if len(sops) == 0 {
			printInfo("No SOPs found")
			return nil
		}

		fmt.Println("SOPs:")
		for _, sop := range sops {
			completed := 0
			for _, step := range sop.Steps {
				if step.Status == "done" {
					completed++
				}
			}
			fmt.Printf("  %s: %s (%d/%d steps complete)\n", sop.ID, sop.Name, completed, len(sop.Steps))
		}
		return nil
	},
}

var sopShowCmd = &cobra.Command{
	Use:   "sop show <id>",
	Short: "Show SOP details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		id := args[0]
		sop, err := dagEngine.GetSOP(id)
		if err != nil {
			return err
		}

		fmt.Printf("SOP: %s\n", sop.Name)
		fmt.Printf("ID: %s\n", sop.ID)
		fmt.Printf("Status: %s\n", sop.Status)
		if sop.Description != "" {
			fmt.Printf("Description: %s\n", sop.Description)
		}
		fmt.Printf("Created: %s\n", sop.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Println("\nSteps:")
		for i, step := range sop.Steps {
			status := string(step.Status)
			if step.Assignee != "" {
				status += fmt.Sprintf(" (assignee: %s)", step.Assignee)
			}
			fmt.Printf("  %d. %s [%s]\n", i+1, step.Name, status)
			if step.Description != "" {
				fmt.Printf("     %s\n", step.Description)
			}
			if len(step.Dependencies) > 0 {
				fmt.Printf("     Dependencies: %v\n", step.Dependencies)
			}
		}
		return nil
	},
}

var sopExecuteCmd = &cobra.Command{
	Use:   "sop execute <id>",
	Short: "Execute an SOP",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		ctx := context.Background()
		id := args[0]

		sop, err := dagEngine.GetSOP(id)
		if err != nil {
			return err
		}

		// Queue ready steps
		if err := dagEngine.QueueReady(id); err != nil {
			return fmt.Errorf("queueing steps: %w", err)
		}

		printSuccess("SOP '%s' execution started (%d steps)", sop.Name, len(sop.Steps))

		// Show which steps are ready
		readySteps, _ := dagEngine.Evaluate(id)
		if len(readySteps) > 0 {
			fmt.Println("\nReady steps:")
			for _, step := range readySteps {
				fmt.Printf("  - %s\n", step.Name)
			}
		}

		// Try to dispatch to available stations
		for _, step := range readySteps {
			stationID, err := plannerInstance.AutoDispatch(ctx, step.ID)
			if err != nil {
				fmt.Printf("  Warning: Could not dispatch step '%s': %v\n", step.Name, err)
				continue
			}
			printInfo("Step '%s' dispatched to station %s", step.Name, stationID)
		}

		return nil
	},
}
