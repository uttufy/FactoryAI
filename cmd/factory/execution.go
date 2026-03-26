package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/uttufy/FactoryAI/internal/beads"
	"github.com/uttufy/FactoryAI/internal/workflow"
)

// getExecutionCmds returns execution commands
func getExecutionCmds() []*cobra.Command {
	return []*cobra.Command{
		runCmd,
		dispatchCmd,
		planCmd,
	}
}

var runCmd = &cobra.Command{
	Use:   "run <job>",
	Short: "Run a job immediately",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		ctx := context.Background()
		jobID := args[0]

		// Verify the job exists
		bead, err := beadsClient.Get(jobID)
		if err != nil {
			return fmt.Errorf("getting job: %w", err)
		}

		// Enqueue for processing
		if err := plannerInstance.Enqueue(ctx, jobID, priority); err != nil {
			return fmt.Errorf("enqueuing job: %w", err)
		}

		printSuccess("Job '%s' queued for execution", bead.Title)

		// Try to auto-dispatch
		stationID, err := plannerInstance.AutoDispatch(ctx, jobID)
		if err != nil {
			printInfo("No available stations. Job queued for later dispatch.")
			return nil
		}

		printInfo("Job dispatched to station %s", stationID)
		return nil
	},
}

var dispatchCmd = &cobra.Command{
	Use:   "dispatch <job> <station>",
	Short: "Dispatch a job to a station",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		ctx := context.Background()
		jobID := args[0]
		stationID := args[1]

		// Verify the job exists
		bead, err := beadsClient.Get(jobID)
		if err != nil {
			return fmt.Errorf("getting job: %w", err)
		}

		// Verify the station exists
		station, err := stationManager.Get(ctx, stationID)
		if err != nil {
			return fmt.Errorf("getting station: %w", err)
		}

		// Dispatch the job
		if err := plannerInstance.Dispatch(ctx, jobID, stationID); err != nil {
			return fmt.Errorf("dispatching job: %w", err)
		}

		printSuccess("Job '%s' dispatched to station %s (%s)", bead.Title, stationID, station.Name)
		return nil
	},
}

var planCmd = &cobra.Command{
	Use:   "plan <goal>",
	Short: "Generate plan from goal",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		goal := args[0]

		// Create a task bead for the goal
		bead, err := beadsClient.Create(string(beads.BeadTask), goal)
		if err != nil {
			return fmt.Errorf("creating task: %w", err)
		}

		// Create a simple SOP from the goal
		steps := []*workflow.Step{
			{
				Name:        "analyze",
				Description: fmt.Sprintf("Analyze the goal: %s", goal),
				Assignee:    "architect",
				Status:      workflow.StepPending,
			},
			{
				Name:         "plan",
				Description:  "Create implementation plan",
				Assignee:     "architect",
				Dependencies: []string{"analyze"},
				Status:       workflow.StepPending,
			},
			{
				Name:         "implement",
				Description:  "Implement the solution",
				Assignee:     "developer",
				Dependencies: []string{"plan"},
				Status:       workflow.StepPending,
			},
			{
				Name:         "review",
				Description:  "Review the implementation",
				Assignee:     "reviewer",
				Dependencies: []string{"implement"},
				Status:       workflow.StepPending,
			},
		}

		sop, err := dagEngine.CreateSOP(goal, steps)
		if err != nil {
			return fmt.Errorf("creating SOP: %w", err)
		}

		printSuccess("Plan created for goal: %s", goal)
		fmt.Printf("SOP ID: %s\n", sop.ID)
		fmt.Println("\nSteps:")
		for i, step := range sop.Steps {
			fmt.Printf("  %d. %s (%s)\n", i+1, step.Name, step.Assignee)
		}

		// Link the bead to the SOP
		printInfo("Task bead created: %s", bead.ID)
		return nil
	},
}
