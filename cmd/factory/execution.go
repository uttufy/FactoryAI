package main

import (
	"github.com/spf13/cobra"
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

		// TODO: Implement job dispatch
		jobID := args[0]
		printSuccess("Job '%s' queued for execution", jobID)
		return nil
	},
}

var dispatchCmd = &cobra.Command{
	Use:   "dispatch <batch>",
	Short: "Dispatch a batch",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		// TODO: Implement batch dispatch
		batchID := args[0]
		printSuccess("Batch '%s' dispatched", batchID)
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

		// TODO: Implement plan generation
		goal := args[0]
		printInfo("Planning for goal: %s", goal)
		return nil
	},
}
